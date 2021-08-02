package board

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	redisClient "github.com/githubixx/pinbackup/redis"

	"github.com/rs/zerolog/log"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// Board contains the raw Pinterest URL, hostname, user, board path,
// path segments and a UUID.
type Board struct {
	RawURL       string   `json:"url"`
	Host         string   `json:"host"`
	User         string   `json:"user"`
	Path         string   `json:"path"`
	PathSegments []string `json:"pathsegments"`
	UUID         string   `json:"uuid"`
}

// respondJSON takes a http.ResponseWriter, a HTTP status code and the
// payload which should be send to the client. payload can be anything
// that can be converted to JSON.
func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	log.Trace().
		Str("method", "respondJSON").
		Msgf("Response to client: %v", payload)

	response, err := json.Marshal(payload)
	if err != nil {
		log.Error().
			Str("method", "respondJSON").
			Msg("Can't create client response. Marshal JSON payload failed.")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(status)
	_, err = w.Write([]byte(response))
	if err != nil {
		log.Error().
			Str("method", "respondJSON").
			Msg("Can't create client response. Writting response failed.")
		return
	}
}

// respondJSON takes a http.ResponseWriter, a HTTP status code and error.
// error is converted to a map containing "error" as key and the string
// representation of the error as value.
func respondError(w http.ResponseWriter, status int, err error) {
	respondJSON(w, status, map[string]string{"error": err.Error()})
}

// trimPath trims "/" at the beginning and at the end of a string.
func trimPath(path string) (string, error) {
	if path == "" {
		return "", errors.New("path to trim can't be empty")
	}

	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	return path, nil
}

// parseUser extracts the username from the URL.
func parseUser(path string) (string, error) {
	var (
		user         string
		splittedPath []string
		err          error
	)

	if path == "" {
		return "", errors.New("Parse user failed: Received empty path")
	}

	path, err = trimPath(path)
	if err != nil {
		return "", errors.New("Parse user failed: Trim '" + path + "' failed")
	}

	if strings.Count(path, "/") < 1 {
		return "", errors.New("Parse user failed: Path doesn't contain at least one /")
	}

	splittedPath = strings.SplitN(path, "/", 2)

	user = splittedPath[0]
	if user == "" {
		return "", errors.New("Parse user failed: Username missing")
	}

	return user, nil
}

// parsePath extracts the path from URL
func parsePath(path string) (string, error) {
	var (
		splittedPath []string
		err          error
	)

	if path == "" {
		return "", errors.New("Parse path failed: Path received is empty")
	}

	path, err = trimPath(path)
	if err != nil {
		return "", errors.New("Parse path failed: Trim '" + path + "' failed")
	}

	splittedPath = strings.SplitN(path, "/", 2)

	path = splittedPath[1]
	if path == "" {
		return "", errors.New("Parse path failed: Path too short")
	}

	return path, nil
}

// parsePathSegments parses the whole URL which either consists only
// of one part or more in case of sections. E.g.:
// https://www.pinterest.de/monivoe/orte-2/
// In the case above "monivoe" is the username and "orte-2" is the
// board name. But also
// https://www.pinterest.de/monivoe/orte-2/subboard
// is possible. In this case the method returns a slice with "orte-2"
// and "subboard" as values.
func parsePathSegments(path string) ([]string, error) {
	var (
		splittedPath []string
		err          error
	)

	if path == "" {
		return make([]string, 0), errors.New("Parse path segments failed: Path received failed")
	}

	path, err = trimPath(path)
	if err != nil {
		return splittedPath, errors.New("Parse path segments failed: Trim '" + path + "' failed")
	}

	splittedPath = strings.Split(path, "/")

	return splittedPath, nil
}

// EnqueueBoard publishes a new board to Redis Pub/Sub. It's the entrypoint
// for every scrape request.
func enqueueBoard(w http.ResponseWriter, r *http.Request) {
	var err error

	log.Trace().
		Str("method", "EnqueueBoard").
		Msg("Getting Redis connection")

	conn, err := redisClient.GetConnection()
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	defer conn.Close()

	// Create new empty board struct
	board := Board{}

	// Decode request
	log.Trace().
		Str("method", "EnqueueBoard").
		Msg("Decoding JSON request")

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&board); err != nil {
		respondError(w, http.StatusBadRequest, errors.New("EnqueueBoard failed: Can't decode JSON request"))
		return
	}
	defer r.Body.Close()
	log.Info().
		Str("method", "EnqueueBoard").
		Msgf("Incoming request: %v", board)

	// Split submitted raw URL in all parts needed later
	parsedURL, err := url.Parse(board.RawURL)
	if err != nil {
		respondError(w, http.StatusBadRequest, errors.New("EnqueueBoard failed: Invalid URL"))
		return
	}

	if parsedURL.Host == "" {
		respondError(w, http.StatusBadRequest, errors.New("EnqueueBoard failed: Invalid hostname"))
		return
	}
	board.Host = parsedURL.Host

	if parsedURL.Path == "" {
		respondError(w, http.StatusBadRequest, errors.New("EnqueueBoard failed: Invalid path"))
		return
	}
	board.Path = parsedURL.Path

	// Extract the username part of the URL
	board.User, err = parseUser(parsedURL.Path)
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	// Extract the path
	board.Path, err = parsePath(parsedURL.Path)
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	// The board path either consists only of one part or more parts in case
	// of sections.
	board.PathSegments, err = parsePathSegments(board.Path)
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	// Generate UUID
	board.UUID = strings.ToLower(uuid.NewV4().String())

	// Convert board to scrape into JSON and publish the request
	// to Redis Pub/Sub to be further processed by the scraper.
	log.Debug().
		Str("method", "EnqueueBoard").
		Msgf("EnqueueBoard: %v", board)

	response, _ := json.Marshal(board)
	if err := redisClient.Publish(conn, "boards", []byte(response)); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	respondJSON(w, http.StatusCreated, board)
}

package board

import (
	"encoding/json"
	"net/http"
	"strings"

	redisClient "github.com/githubixx/pinbackup/redis"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type requestPath struct {
	Path string `json:"path"`
}

type boardExists struct {
	Exists bool   `json:"exists"`
	Path   string `json:"path"`
}

// prepareKey expects a path like "/user/board/" as input. This input is
// used to generate the key for the board to store it in Redis.
func prepareKey(path string) (string, error) {
	if path == "" {
		return "", errors.New("prepareKey failed: Received empty path")
	}

	trimmedPath, err := trimPath(path)
	if err != nil {
		return "", err
	}

	if strings.Count(path, "/") < 1 {
		return "", errors.New("prepareKey failed: Path doesn't contain at least one /")
	}

	key := strings.Replace(trimmedPath, "/", ":", 1)

	return key, nil
}

// existsBoard verifies if a board is already stored in Redis.
func existsBoard(w http.ResponseWriter, r *http.Request) {
	var err error

	log.Trace().
		Str("method", "existsBoard").
		Msg("Getting Redis connection")

	conn, err := redisClient.GetConnection()
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	defer conn.Close()

	requestPath := requestPath{}

	log.Trace().
		Str("method", "existsBoard").
		Msg("Decoding JSON request")

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&requestPath); err != nil {
		respondError(w, http.StatusBadRequest, errors.New("existsBoard failed: Can't decode JSON request"))
		return
	}
	defer r.Body.Close()
	log.Info().
		Str("method", "existsBoard").
		Msgf("Incoming request: %v", requestPath)

	key, err := prepareKey(requestPath.Path)
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	exists, err := redisClient.ExistsBoard(conn, key)
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	j := boardExists{Exists: exists, Path: requestPath.Path}

	respondJSON(w, http.StatusOK, j)

}

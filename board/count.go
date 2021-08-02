package board

import (
	"encoding/json"
	"net/http"

	redisClient "github.com/githubixx/pinbackup/redis"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type boardCount struct {
	Count int `json:"count"`
}

func countBoard(w http.ResponseWriter, r *http.Request) {
	var err error

	log.Trace().
		Str("method", "countBoard").
		Msg("Getting Redis connection")

	conn, err := redisClient.GetConnection()
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	defer conn.Close()

	requestPath := requestPath{}

	log.Trace().
		Str("method", "countBoard").
		Msg("Decoding JSON request")

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&requestPath); err != nil {
		respondError(w, http.StatusBadRequest, errors.New("countBoard failed: Can't decode JSON request"))
		return
	}
	defer r.Body.Close()
	log.Info().
		Str("method", "countBoard").
		Msgf("Incoming request: %v", requestPath)

	key, err := prepareKey(requestPath.Path)
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	count, err := redisClient.CountPictures(conn, key)
	if err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	j := boardCount{Count: count}

	respondJSON(w, http.StatusOK, j)

}

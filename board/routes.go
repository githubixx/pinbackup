package board

import (
	"github.com/gorilla/mux"
)

// Routes /api entry point
func Routes() *mux.Router {
	router := mux.NewRouter()

	subRouter := router.PathPrefix("/api").Subrouter()
	subRouter.HandleFunc("/v1/board", EnqueueBoard).Methods("POST")

	return router
}

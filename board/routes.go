package board

import (
	"github.com/gorilla/mux"
)

// Routes /api entry point
func Routes() *mux.Router {
	router := mux.NewRouter()

	subRouter := router.PathPrefix("/api").Subrouter()
	subRouter.HandleFunc("/v1/board", enqueueBoard).Methods("POST")
	subRouter.HandleFunc("/v1/existsboard", existsBoard).Methods("POST")
	subRouter.HandleFunc("/v1/countboard", countBoard).Methods("POST")

	return router
}

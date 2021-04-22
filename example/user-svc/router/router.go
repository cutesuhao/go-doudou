package router

import (
	"github.com/gorilla/mux"
	"net/http"
)

type route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

func NewRouter() *mux.Router {
	rous := routes()
	router := mux.NewRouter().StrictSlash(true)
	for _, r := range rous {
		var handler http.Handler

		handler = r.HandlerFunc
		handler = logger(handler, r.Name)
		handler = rest(handler)

		router.
			Methods(r.Method).
			Path(r.Pattern).
			Name(r.Name).
			Handler(handler)
	}
	return router
}
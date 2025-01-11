package main

import (
	"log"
	"net/http"

	"github.com/NomadXD/flowc/pkg/filters"
)

type HeaderFilter struct{}

func (m *HeaderFilter) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("X-Flowc-Example-request-flow", "true")
		w.Header().Set("X-Flowc-Example-response-flow", "true")
		log.Printf("Headers added")
		next.ServeHTTP(w, r)
	})
}

// NewFilter is the entry point for this plugin.
func NewFilter() filters.HTTPFilter {
	return &HeaderFilter{}
}

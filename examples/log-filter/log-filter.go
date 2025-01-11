package main

import (
	"log"
	"net/http"

	"github.com/NomadXD/flowc/pkg/filters"
)

type ExampleFilter struct{}

func (m *ExampleFilter) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Print("Before Example Middleware")
		next.ServeHTTP(w, r)
		log.Print("After Example Middleware")
	})
}

// NewFilter is the entry point for this plugin.
func NewFilter() filters.HTTPFilter {
	return &ExampleFilter{}
}

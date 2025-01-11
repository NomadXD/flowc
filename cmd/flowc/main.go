package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/NomadXD/flowc/config"
	"github.com/NomadXD/flowc/internal/filters"
	"github.com/NomadXD/flowc/internal/proxy"
)

func main() {
	config := config.Config{}
	configFile, err := os.Open("config.json")
	if err != nil {
		log.Fatalf("Failed to open config file: %v", err)
	}
	defer configFile.Close()

	if err := json.NewDecoder(configFile).Decode(&config); err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	proxy := proxy.NewSingleHostReverseProxy(&config.UpstreamConfig)
	filters := filters.FilterChain{}
	for _, mw := range config.Filters {
		if err := filters.LoadFromPlugin(mw.Path); err != nil {
			log.Fatalf("Failed to load middleware from %s: %v", mw.Path, err)
		}
	}
	handler := filters.ConstructFilterChain(proxy)
	listenAddress := ":8080"
	if err := http.ListenAndServe(listenAddress, handler); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}

}

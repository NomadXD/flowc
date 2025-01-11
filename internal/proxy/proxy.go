package proxy

import (
	"log"
	"net/http/httputil"
	"net/url"

	"github.com/NomadXD/flowc/config"
)

func NewSingleHostReverseProxy(config *config.UpstreamConfig) *httputil.ReverseProxy {
	if err := validateConfig(config); err != nil {
		panic(err)
	}
	parsedURL, _ := url.Parse(config.Target)
	log.Printf("Proxying to %s\n", parsedURL)
	return httputil.NewSingleHostReverseProxy(parsedURL)
}

func validateConfig(config *config.UpstreamConfig) error {
	if _, err := url.Parse(config.Target); err != nil {
		return err
	}
	return nil
}

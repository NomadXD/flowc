package filters

import "net/http"

type HTTPFilter interface {
	Handle(next http.Handler) http.Handler
}

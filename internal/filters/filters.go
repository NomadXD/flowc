package filters

import (
	"errors"
	"net/http"
	"plugin"

	"github.com/NomadXD/flowc/pkg/filters"
)

type FilterChain struct {
	filters []filters.HTTPFilter
}

func (fc *FilterChain) Add(filter filters.HTTPFilter) {
	fc.filters = append(fc.filters, filter)
}

// LoadFromPlugin loads a HTTPFilter from a shared object file.
func (fc *FilterChain) LoadFromPlugin(path string) error {
	p, err := plugin.Open(path)
	if err != nil {
		return err
	}

	symbol, err := p.Lookup("NewFilter")
	if err != nil {
		return errors.New("plugin does not define NewMiddleware function")
	}

	// NewFilter should return a HTTPFilter implementation
	constructor, ok := symbol.(func() filters.HTTPFilter)
	if !ok {
		return errors.New("invalid NewMiddleware signature")
	}

	fc.Add(constructor())
	return nil
}

func (fc *FilterChain) ConstructFilterChain(upstreamHandler http.Handler) http.Handler {
	handler := upstreamHandler
	for _, filter := range fc.filters { // Apply filters in order
		handler = filter.Handle(handler)
	}
	return handler
}

package backend

import (
	"fmt"
	"github.com/valyala/fasthttp"
)

var Backends map[string]Backend

type Backend interface {
	Invoke(string, *fasthttp.Request, *fasthttp.Response) error
}

func AddBackend(name string, backend Backend) {
	Backends[name] = backend
}

func GetBackend(name string) (Backend, error) {
	bke, ok := Backends[name]
	if !ok {
		return nil, fmt.Errorf("[backend] no request backend %s\n", name)
	}

	return bke, nil
}
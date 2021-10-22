package backend

import (
	"fmt"
	"github.com/valyala/fasthttp"
)

const yuanrongBackendName = "yuanrong"

type YuanrongBackend struct {
	Server string
}

func (y *YuanrongBackend) Invoke(target string, req *fasthttp.Request, res *fasthttp.Response) error {
	uri := fmt.Sprintf("http://%s/serverless/v1/functions/%s/invocations",
		y.Server, target)
	fmt.Printf("yuanrong uri %s\n", uri)

	req.SetRequestURI(uri)
	return fasthttp.Do(req, res)
}

func NewYuanrongBackend(server string) {
	yBackend := &YuanrongBackend{
		Server: server,
	}
	AddBackend(yuanrongBackendName, yBackend)
}
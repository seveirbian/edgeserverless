package backend

import "github.com/valyala/fasthttp"

const yuanrongBackendName = "yuanrong"

type YuanrongBackend struct {
	Server string
}

func (y *YuanrongBackend) Invoke(target string, req *fasthttp.Request, res *fasthttp.Response) error {
	return nil
}

func NewYuanrongBackend(server string) {
	yBackend := &YuanrongBackend{
		Server: server,
	}
	AddBackend(yuanrongBackendName, yBackend)
}
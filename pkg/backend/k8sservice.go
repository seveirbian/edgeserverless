package backend

import "github.com/valyala/fasthttp"

const k8sServiceBackendName = "k8sservice"

type K8sServiceBackend struct {
}

func (k *K8sServiceBackend) Invoke(target string, req *fasthttp.Request, res *fasthttp.Response) error {
	req.SetRequestURI(target)
	return fasthttp.Do(req, res)
}

func NewK8sServiceBackend() {
	AddBackend(k8sServiceBackendName, &K8sServiceBackend{})
}

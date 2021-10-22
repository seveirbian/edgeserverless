package entry

import (
	"fmt"
	fiber "github.com/gofiber/fiber/v2"
	wr "github.com/mroth/weightedrand"
	v1alpha1 "github.com/seveirbian/edgeserverless/pkg/apis/edgeserverless/v1alpha1"
	"github.com/seveirbian/edgeserverless/pkg/backend"
	"github.com/seveirbian/edgeserverless/pkg/rulesmanager"
	"github.com/valyala/fasthttp"
	"math/rand"
	"time"
)

type Entry struct {
	Server *fiber.App
	Addr string

	RulesManager *rulesmanager.RulesManager
	HTTPClient *fasthttp.Client
}

func NewEntry(rulesManager *rulesmanager.RulesManager) *Entry {
	app := fiber.New()

	client := &fasthttp.Client{
		NoDefaultUserAgentHeader: true,
		DisablePathNormalizing:   true,
	}

	return &Entry{
		Server: app,
		Addr: ":1122",
		RulesManager: rulesManager,
		HTTPClient: client,
	}
}

func (e *Entry) Start () {
	e.Server.Get("/*", e.serve)
	e.Server.Post("/*", e.serve)

	err := e.Server.Listen(e.Addr)
	if err != nil {
		fmt.Printf("[entry] exit with %v\n", err)
	}
}

func (e *Entry) serve (c *fiber.Ctx) error {
	uri := c.Hostname() + c.Path()
	fmt.Println(uri)
	route, err := e.RulesManager.GetRule(uri)
	if err != nil {
		return c.SendString(fmt.Sprintf("error %v\n", err))
	}

	rand.Seed(time.Now().UTC().UnixNano())

	choices := []wr.Choice{}
	for _, t := range route.Targets {
		choices = append(choices, wr.Choice{Item: t,
			Weight: uint(t.Ratio)})
	}

	chooser, _ := wr.NewChooser(choices...)

	target := chooser.Pick().(v1alpha1.RouteTarget)
	fmt.Println(target)

	req := c.Request()
	res := c.Response()

	bke, err := backend.GetBackend(target.Type)
	if err != nil {
		c.Response().SetStatusCode(500)
	}

	return bke.Invoke(target.Target, req, res)
}
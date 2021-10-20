package entry

import (
	"fmt"
	fiber "github.com/gofiber/fiber/v2"
	"github.com/seveirbian/edgeserverless/pkg/rulesmanager"
	"github.com/valyala/fasthttp"
	wr "github.com/mroth/weightedrand"
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
	e.Server.Get("/*", func(c *fiber.Ctx) error {
		uri := c.Hostname() + c.Path()
		fmt.Println(uri)
		route, err := e.RulesManager.GetRule(uri)
		if err != nil {
			return c.SendString(fmt.Sprintf("error %v\n", err))
		}

		rand.Seed(time.Now().UTC().UnixNano())

		choices := []wr.Choice{}
		for _, target := range route.Targets {
			choices = append(choices, wr.Choice{Item: target.FunctionUrn,
				Weight: uint(target.Ratio)})
		}

		chooser, _ := wr.NewChooser(choices...)

		result := chooser.Pick().(string)
		fmt.Println(result)

		req := c.Request()
		res := c.Response()

		req.SetRequestURI(result)

		return e.HTTPClient.Do(req, res)
	})

	e.Server.Listen(e.Addr)
}
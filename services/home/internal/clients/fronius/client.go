package fronius

import (
	"net/http"
	"net/url"
	"os"
	"time"
)

var baseUrl = url.URL{
	Scheme: "http",
	Host:   os.Getenv("FRONIUS_PV_HOST"),
	Path:   "/solar_api/v1/",
}

func New() *client {
	c := &http.Client{Timeout: 2 * time.Second}
	return &client{
		httpClient: c,
	}
}

type client struct {
	httpClient *http.Client
}

type response[T any] struct {
	Body T
}

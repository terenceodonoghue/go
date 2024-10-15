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

func New() *Client {
	c := &http.Client{Timeout: 2 * time.Second}
	return &Client{
		httpClient: c,
	}
}

type Client struct {
	httpClient *http.Client
}

type Response[T any] struct {
	Body T
}

package sensibo

import (
	"net/http"
	"net/url"
	"os"
	"time"
)

var baseUrl = url.URL{
	Scheme: "https",
	Host:   os.Getenv("SENSIBO_AC_HOST"),
	Path:   "/api/v2/",
}

func New(apiKey string) *client {
	c := &http.Client{Timeout: 10 * time.Second}
	return &client{
		apiKey:     apiKey,
		httpClient: c,
	}
}

type client struct {
	apiKey     string
	httpClient *http.Client
}

type response[T any] struct {
	Status string
	Result []T
}

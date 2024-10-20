package fronius

import (
	"encoding/json"
	"net/http"
	"net/url"
)

func (c *client) GetLoggerInfo(ch chan<- Logger) error {
	defer close(ch)
	rel := &url.URL{Path: "GetLoggerInfo.cgi"}
	url := baseUrl.ResolveReference(rel)

	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return err
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	var response response[Logger]
	err = json.NewDecoder(res.Body).Decode(&response)
	ch <- response.Body
	return err
}

type Logger struct {
	LoggerInfo struct {
		CO2Factor float32
		CO2Unit   string
	}
}

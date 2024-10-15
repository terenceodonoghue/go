package fronius

import (
	"encoding/json"
	"net/http"
	"net/url"
)

func (c *Client) GetRealtimeData(pv chan<- Inverter) error {
	rel := &url.URL{Path: "GetInverterRealtimeData.cgi"}
	url := baseUrl.ResolveReference(rel)
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return err
	}

	q := req.URL.Query()

	q.Add("Scope", "System")
	req.URL.RawQuery = q.Encode()

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	var response Response[Inverter]
	err = json.NewDecoder(res.Body).Decode(&response)
	pv <- response.Body
	return err
}

func (i Inverter) MarshalJSON() ([]byte, error) {
	inverter := struct {
		Power        Output `json:"power"`
		DailyEnergy  Output `json:"daily_energy"`
		AnnualEnergy Output `json:"annual_energy"`
		TotalEnergy  Output `json:"total_energy"`
	}{
		Power:        i.Data.PAC,
		DailyEnergy:  i.Data.DAY_ENERGY,
		AnnualEnergy: i.Data.YEAR_ENERGY,
		TotalEnergy:  i.Data.TOTAL_ENERGY,
	}

	return json.Marshal(inverter)
}

func (o Output) MarshalJSON() ([]byte, error) {
	output := struct {
		Value int    `json:"value"`
		Unit  string `json:"unit"`
	}{
		Value: o.Values.Sum(),
		Unit:  o.Unit,
	}

	return json.Marshal(output)
}

func (v Values) Sum() int {
	sum := 0
	for _, t := range v {
		sum += t
	}

	return sum
}

type Inverter struct {
	Data struct {
		PAC          Output
		DAY_ENERGY   Output
		YEAR_ENERGY  Output
		TOTAL_ENERGY Output
	}
}

type Output struct {
	Unit   string
	Values Values
}

type Values map[string]int

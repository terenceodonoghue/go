package fronius

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// StatusRunning is the Fronius StatusCode for a normally operating inverter.
const StatusRunning = 7

// InverterValue is a unit+value pair returned by the Fronius Solar API.
type InverterValue struct {
	Unit  string  `json:"Unit"`
	Value float64 `json:"Value"`
}

// DeviceStatus holds operational status fields.
type DeviceStatus struct {
	ErrorCode  int `json:"ErrorCode"`
	StatusCode int `json:"StatusCode"`
}

// CommonInverterData holds all fields from DataCollection=CommonInverterData.
type CommonInverterData struct {
	PAC          InverterValue `json:"PAC"`
	IAC          InverterValue `json:"IAC"`
	UAC          InverterValue `json:"UAC"`
	FAC          InverterValue `json:"FAC"`
	IDC          InverterValue `json:"IDC"`
	UDC          InverterValue `json:"UDC"`
	DayEnergy    InverterValue `json:"DAY_ENERGY"`
	YearEnergy   InverterValue `json:"YEAR_ENERGY"`
	TotalEnergy  InverterValue `json:"TOTAL_ENERGY"`
	DeviceStatus DeviceStatus  `json:"DeviceStatus"`
}

// RealtimeDataResponse is the full envelope from the Fronius Solar API.
type RealtimeDataResponse struct {
	Body struct {
		Data CommonInverterData `json:"Data"`
	} `json:"Body"`
	Head struct {
		Status struct {
			Code int `json:"Code"`
		} `json:"Status"`
	} `json:"Head"`
}

// Client is an HTTP client scoped to a single Fronius inverter.
type Client struct {
	baseURL       string
	httpClient    *http.Client
	archiveClient *http.Client
}

// New returns a Client targeting the given inverter base URL (e.g., "http://192.168.1.100").
// The provided http.Client controls timeout behaviour for realtime requests; set a timeout
// shorter than the poll interval (e.g., 4 seconds) to avoid pile-up on slow responses.
// Archive requests use a separate client with a longer timeout (60 seconds).
func New(baseURL string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:       baseURL,
		httpClient:    httpClient,
		archiveClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// Fetch retrieves CommonInverterData from the inverter. It returns an error if the HTTP
// request fails, the response cannot be decoded, or the API-level status code is non-zero.
// Check DeviceStatus.StatusCode before writing metrics — a successful fetch does not imply
// the inverter is producing power.
func (c *Client) Fetch(ctx context.Context) (*RealtimeDataResponse, error) {
	url := c.baseURL + "/solar_api/v1/GetInverterRealtimeData.cgi?Scope=Device&DeviceId=1&DataCollection=CommonInverterData"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("fronius: build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fronius: do request: %w", err)
	}
	defer resp.Body.Close()

	var result RealtimeDataResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("fronius: decode response: %w", err)
	}

	if result.Head.Status.Code != 0 {
		return nil, fmt.Errorf("fronius: API error code %d", result.Head.Status.Code)
	}

	return &result, nil
}

// archiveChannelData holds time-series values for one data channel.
type archiveChannelData struct {
	Unit   string             `json:"Unit"`
	Values map[string]float64 `json:"Values"`
}

// archiveDeviceData holds channel data for one device in the archive response.
type archiveDeviceData struct {
	Data map[string]archiveChannelData `json:"Data"`
}

// archiveResponse is the full envelope from GetArchiveData.
type archiveResponse struct {
	Body struct {
		Data map[string]archiveDeviceData `json:"Data"`
	} `json:"Body"`
	Head struct {
		Status struct {
			Code int `json:"Code"`
		} `json:"Status"`
	} `json:"Head"`
}

// FetchMonthEnergy returns the total energy produced in the current calendar
// month (Wh) by summing daily values from GetArchiveData. Unlike the running
// total_energy difference stored in InfluxDB, this reflects the inverter's own
// historical records regardless of when this service started running.
//
// The inverter limits archive queries to 15 days, so months longer than that
// are fetched in chunks and summed.
func (c *Client) FetchMonthEnergy(ctx context.Context, now time.Time) (float64, error) {
	const chunkDays = 15

	var total float64
	chunkStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	for !chunkStart.After(now) {
		chunkEnd := chunkStart.AddDate(0, 0, chunkDays-1)
		if chunkEnd.After(now) {
			chunkEnd = now
		}
		wh, err := c.fetchArchiveChunk(ctx, chunkStart, chunkEnd)
		if err != nil {
			return 0, err
		}
		total += wh
		chunkStart = chunkStart.AddDate(0, 0, chunkDays)
	}
	return total, nil
}

func (c *Client) fetchArchiveChunk(ctx context.Context, start, end time.Time) (float64, error) {
	url := fmt.Sprintf(
		"%s/solar_api/v1/GetArchiveData.cgi?Scope=Device&DeviceClass=Inverter&DeviceId=1&StartDate=%s&EndDate=%s&Channel=EnergyReal_WAC_Sum_Produced&SeriesType=DailySum",
		c.baseURL,
		start.Format("02.01.2006"),
		end.Format("02.01.2006"),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("fronius: build archive request: %w", err)
	}

	resp, err := c.archiveClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("fronius: do archive request: %w", err)
	}
	defer resp.Body.Close()

	var result archiveResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("fronius: decode archive response: %w", err)
	}

	if result.Head.Status.Code != 0 {
		return 0, fmt.Errorf("fronius: archive API error code %d", result.Head.Status.Code)
	}

	var total float64
	for _, device := range result.Body.Data {
		if ch, ok := device.Data["EnergyReal_WAC_Sum_Produced"]; ok {
			for _, v := range ch.Values {
				total += v
			}
		}
	}
	return total, nil
}

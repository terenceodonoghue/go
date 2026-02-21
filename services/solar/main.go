package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"go.local/services/solar/internal/fronius"
)

const (
	deviceID        = "fronius"
	healthAddr      = ":8082"
	pollInterval    = 5 * time.Second
	backoffMax      = 10 * time.Minute
	archiveInterval = 24 * time.Hour
)

func main() {
	inverterURL := requiredEnv("INVERTER_URL")
	capacityW, err := strconv.ParseFloat(requiredEnv("INVERTER_CAPACITY_W"), 64)
	if err != nil {
		log.Fatalf("Invalid INVERTER_CAPACITY_W: %v", err)
	}

	influxURL    := requiredEnv("INFLUX_URL")
	influxToken  := requiredEnv("INFLUX_TOKEN")
	influxOrg    := requiredEnv("INFLUX_ORG")
	influxBucket := requiredEnv("INFLUX_BUCKET")

	froniusClient := fronius.New(inverterURL, &http.Client{Timeout: 4 * time.Second})

	influxClient := influxdb2.NewClient(influxURL, influxToken)
	defer influxClient.Close()
	writeAPI := influxClient.WriteAPIBlocking(influxOrg, influxBucket)

	log.Println("Configuration:")
	log.Printf("  INVERTER_URL        = %s", inverterURL)
	log.Printf("  INVERTER_CAPACITY_W = %.0f", capacityW)
	log.Printf("  INFLUX_URL          = %s", influxURL)
	log.Printf("  INFLUX_ORG          = %s", influxOrg)
	log.Printf("  INFLUX_BUCKET       = %s", influxBucket)
	log.Println()

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		log.Printf("Health check listening on %s", healthAddr)
		if err := http.ListenAndServe(healthAddr, mux); err != nil {
			log.Fatalf("Health check server failed: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	interval := pollInterval
	timer := time.NewTimer(0) // fire immediately on start
	defer timer.Stop()

	archiveTimer := time.NewTimer(0) // fire immediately on start
	defer archiveTimer.Stop()

	var offline bool
	var cachedMonthEnergy float64

	log.Printf("Polling inverter every %s (backoff max %s)", pollInterval, backoffMax)
	log.Printf("Polling archive every %s", archiveInterval)

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down")
			return
		case <-archiveTimer.C:
			if wh, err := froniusClient.FetchMonthEnergy(ctx, time.Now()); err != nil {
				if ctx.Err() == nil {
					log.Printf("Archive fetch failed: %v", err)
				}
			} else {
				cachedMonthEnergy = wh
			}
			if ctx.Err() == nil {
				archiveTimer.Reset(archiveInterval)
			}
		case <-timer.C:
			err := poll(ctx, froniusClient, writeAPI, deviceID, capacityW, cachedMonthEnergy)
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				if !offline {
					log.Printf("Inverter unreachable: %v", err)
					offline = true
				}
				interval = min(interval*2, backoffMax)
				log.Printf("Retrying in %s", interval)
			} else {
				if offline {
					log.Println("Inverter back online, resuming normal polling")
					offline = false
				}
				interval = pollInterval
			}
			timer.Reset(interval)
		}
	}
}

func poll(ctx context.Context, client *fronius.Client, writeAPI api.WriteAPIBlocking, deviceID string, capacityW float64, monthEnergyWh float64) error {
	data, err := client.Fetch(ctx)
	if err != nil {
		return err
	}

	if data.Body.Data.DeviceStatus.StatusCode != fronius.StatusRunning {
		return nil
	}

	d := data.Body.Data
	pacW := d.PAC.Value

	fields := map[string]interface{}{
		"pac":          pacW,
		"iac":          d.IAC.Value,
		"uac":          d.UAC.Value,
		"fac":          d.FAC.Value,
		"idc":          d.IDC.Value,
		"udc":          d.UDC.Value,
		"day_energy":   d.DayEnergy.Value,
		"year_energy":  d.YearEnergy.Value,
		"total_energy": d.TotalEnergy.Value,
		"pac_kw":       pacW / 1000,
		"utilisation":  (pacW / capacityW) * 100,
	}
	if monthEnergyWh > 0 {
		fields["month_energy"] = monthEnergyWh
	}

	p := influxdb2.NewPoint(
		"inverter",
		map[string]string{"device_id": deviceID},
		fields,
		time.Now(),
	)

	return writeAPI.WritePoint(ctx, p)
}

func requiredEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s is required", key)
	}
	return v
}

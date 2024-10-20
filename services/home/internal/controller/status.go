package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/terenceodonoghue/go/services/home/internal/clients/fronius"
	"github.com/terenceodonoghue/go/services/home/internal/clients/sensibo"
	"golang.org/x/sync/errgroup"
)

func GetStatus(ctx context.Context) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		errs, ctx := errgroup.WithContext(ctx)

		f := fronius.New()
		s := sensibo.New(os.Getenv("SENSIBO_API_KEY"))

		ac := make(chan []sensibo.Device, 1)
		pv := make(chan fronius.Inverter, 1)

		errs.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return s.GetDevices(ac)
			}
		})

		errs.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return f.GetInverterRealtimeData(pv)
			}
		})

		if err := errs.Wait(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		p := map[string]interface{}{
			"ac": <-ac,
			"pv": <-pv,
		}

		json.NewEncoder(w).Encode(p)
	})
}

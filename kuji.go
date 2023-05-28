// original here - https://gist.github.com/jumanzii/031cfea1b2aa3c2a43b63aa62a919285
package main

import (
	"context"
	"net/http"
	"sync"
	"time"

	oracletypes "github.com/Team-Kujira/core/x/oracle/types"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

/*
type voteMissCounter struct {
	MissCount string `json:"miss_count"`
}
*/

func KujiraMetricHandler(w http.ResponseWriter, r *http.Request, grpcConn *grpc.ClientConn) {
	requestStart := time.Now()

	sublogger := log.With().
		Str("request-id", uuid.New().String()).
		Logger()

	address := r.URL.Query().Get("address")

	votePenaltyCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "cosmos_kujira_oracle_vote_miss_count",
			Help:        "Vote miss count",
			ConstLabels: ConstLabels,
		},
		[]string{"type"},
	)

	registry := prometheus.NewRegistry()
	registry.MustRegister(votePenaltyCount)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started querying oracle feeder metrics")
		queryStart := time.Now()

		oracleClient := oracletypes.NewQueryClient(grpcConn)
		response, err := oracleClient.MissCounter(context.Background(), &oracletypes.QueryMissCounterRequest{ValidatorAddr: address})

		if err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not get oracle feeder metrics")
			return
		}

		sublogger.Debug().
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying oracle feeder metrics")

		missCount := float64(response.MissCounter)

		votePenaltyCount.WithLabelValues("miss").Add(missCount)

	}()
	wg.Wait()

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
	sublogger.Info().
		Str("method", "GET").
		Str("endpoint", "/metrics/kujira").
		Float64("request-time", time.Since(requestStart).Seconds()).
		Msg("Request processed")
}

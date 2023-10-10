// original here - https://gist.github.com/jumanzii/031cfea1b2aa3c2a43b63aa62a919285
package main

import (
	"context"
	oracletypes "github.com/Team-Kujira/core/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/rs/zerolog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

/*
	type voteMissCounter struct {
		MissCount string `json:"miss_count"`
	}
*/
type KujiMetrics struct {
	votePenaltyCount *prometheus.CounterVec
}

func NewKujiMetrics(reg prometheus.Registerer) *KujiMetrics {
	m := &KujiMetrics{
		votePenaltyCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "cosmos_kujira_oracle_vote_miss_count",
				Help:        "Vote miss count",
				ConstLabels: ConstLabels,
			},
			[]string{"type"},
		),
	}

	reg.MustRegister(m.votePenaltyCount)

	return m
}
func getKujiMetrics(wg *sync.WaitGroup, sublogger *zerolog.Logger, metrics *KujiMetrics, s *service, validatorAddress sdk.ValAddress) {
	wg.Add(1)

	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started querying oracle feeder metrics")
		queryStart := time.Now()

		oracleClient := oracletypes.NewQueryClient(s.grpcConn)
		response, err := oracleClient.MissCounter(context.Background(), &oracletypes.QueryMissCounterRequest{ValidatorAddr: validatorAddress.String()})

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

		metrics.votePenaltyCount.WithLabelValues("miss").Add(missCount)

	}()
}
func (s *service) KujiraMetricHandler(w http.ResponseWriter, r *http.Request) {
	requestStart := time.Now()

	sublogger := log.With().
		Str("request-id", uuid.New().String()).
		Logger()

	address := r.URL.Query().Get("address")
	myAddress, err := sdk.ValAddressFromBech32(address)
	if err != nil {
		sublogger.Error().
			Str("address", address).
			Err(err).
			Msg("Could not get address")
		return
	}
	registry := prometheus.NewRegistry()
	kujiMetrics := NewKujiMetrics(registry)

	var wg sync.WaitGroup
	getKujiMetrics(&wg, &sublogger, kujiMetrics, s, myAddress)

	wg.Wait()

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
	sublogger.Info().
		Str("method", "GET").
		Str("endpoint", "/metrics/kujira").
		Float64("request-time", time.Since(requestStart).Seconds()).
		Msg("Request processed")
}

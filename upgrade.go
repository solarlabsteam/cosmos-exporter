package main

import (
	"context"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/rs/zerolog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type UpgradeMetrics struct {
	upgradePlanGauge *prometheus.GaugeVec
}

func NewUpgradeMetrics(reg prometheus.Registerer) *UpgradeMetrics {
	m := &UpgradeMetrics{
		upgradePlanGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "cosmos_upgrade_plan",
				Help:        "Upgrade plan info in height",
				ConstLabels: ConstLabels,
			},
			[]string{"info", "name", "height", "estimated_time"},
		),
	}
	reg.MustRegister(m.upgradePlanGauge)
	return m
}
func getUpgradeMetrics(wg *sync.WaitGroup, sublogger *zerolog.Logger, metrics *UpgradeMetrics, s *service) {

	wg.Add(1)
	go func() {
		defer wg.Done()
		queryStart := time.Now()

		upgradeClient := upgradetypes.NewQueryClient(s.grpcConn)
		upgradeRes, err := upgradeClient.CurrentPlan(
			context.Background(),
			&upgradetypes.QueryCurrentPlanRequest{},
		)
		if err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not get upgrade plan")
			return
		}

		sublogger.Debug().
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying upgrade plan")

		if upgradeRes.Plan == nil {
			metrics.upgradePlanGauge.With(prometheus.Labels{
				"info":           "None",
				"name":           "None",
				"height":         "",
				"estimated_time": "",
			}).Set(0)
			return
		}

		cs, err := NewChainStatus()
		if err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not get sync info")
			return
		}

		upgradeHeight := upgradeRes.Plan.Height
		remainingHeight := upgradeHeight - cs.LatestBlockHeight()

		if remainingHeight <= 0 {
			metrics.upgradePlanGauge.With(prometheus.Labels{
				"info":           "None",
				"name":           "None",
				"height":         "",
				"estimated_time": "",
			}).Set(0)
			return
		}

		estimatedTime, err := cs.EstimateBlockTime(remainingHeight)
		if err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not get estimated time")
		}

		metrics.upgradePlanGauge.With(prometheus.Labels{
			"info":           upgradeRes.Plan.Info,
			"name":           upgradeRes.Plan.Name,
			"height":         strconv.FormatInt(upgradeHeight, 10),
			"estimated_time": estimatedTime.Local().Format(time.RFC1123),
		}).Set(float64(remainingHeight))
	}()

}
func (s *service) UpgradeHandler(w http.ResponseWriter, r *http.Request) {
	requestStart := time.Now()

	sublogger := log.With().
		Str("request-id", uuid.New().String()).
		Logger()

	registry := prometheus.NewRegistry()
	upgradeMetrics := NewUpgradeMetrics(registry)

	var wg sync.WaitGroup
	getUpgradeMetrics(&wg, &sublogger, upgradeMetrics, s)

	wg.Wait()

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
	sublogger.Info().
		Str("method", "GET").
		Str("endpoint", "/metrics/upgrade").
		Float64("request-time", time.Since(requestStart).Seconds()).
		Msg("Request processed")
}

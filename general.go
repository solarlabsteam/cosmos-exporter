package main

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func GeneralHandler(w http.ResponseWriter, r *http.Request, grpcConn *grpc.ClientConn) {
	requestStart := time.Now()

	sublogger := log.With().
		Str("request-id", uuid.New().String()).
		Logger()

	generalBondedTokensGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "cosmos_general_bonded_tokens",
			Help: "Bonded tokens",
		},
	)

	generalNotBondedTokensGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "cosmos_general_not_bonded_tokens",
			Help: "Not bonded tokens",
		},
	)

	generalCommunityPoolGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_general_community_pool",
			Help: "Community pool",
		},
		[]string{"denom"},
	)

	generalSupplyTotalGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_general_supply_total",
			Help: "Total supply",
		},
		[]string{"denom"},
	)

	registry := prometheus.NewRegistry()
	registry.MustRegister(generalBondedTokensGauge)
	registry.MustRegister(generalNotBondedTokensGauge)
	registry.MustRegister(generalCommunityPoolGauge)
	registry.MustRegister(generalSupplyTotalGauge)

	var wg sync.WaitGroup

	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started querying staking pool")
		queryStart := time.Now()

		stakingClient := stakingtypes.NewQueryClient(grpcConn)
		response, err := stakingClient.Pool(
			context.Background(),
			&stakingtypes.QueryPoolRequest{},
		)
		if err != nil {
			sublogger.Error().Err(err).Msg("Could not get staking pool")
			return
		}

		sublogger.Debug().
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying staking pool")

		generalBondedTokensGauge.Set(float64(response.Pool.BondedTokens.Int64()))
		generalNotBondedTokensGauge.Set(float64(response.Pool.NotBondedTokens.Int64()))
	}()
	wg.Add(1)

	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started querying distribution community pool")
		queryStart := time.Now()

		distributionClient := distributiontypes.NewQueryClient(grpcConn)
		response, err := distributionClient.CommunityPool(
			context.Background(),
			&distributiontypes.QueryCommunityPoolRequest{},
		)
		if err != nil {
			sublogger.Error().Err(err).Msg("Could not get distribution community pool")
			return
		}

		sublogger.Debug().
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying distribution community pool")

		for _, coin := range response.Pool {
			if value, err := strconv.ParseFloat(coin.Amount.String(), 64); err != nil {
				sublogger.Error().
					Err(err).
					Msg("Could not get community pool coin")
			} else {
				generalCommunityPoolGauge.With(prometheus.Labels{
					"denom": coin.Denom,
				}).Set(value)
			}
		}
	}()
	wg.Add(1)

	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started querying bank total supplyy")
		queryStart := time.Now()

		bankClient := banktypes.NewQueryClient(grpcConn)
		response, err := bankClient.TotalSupply(
			context.Background(),
			&banktypes.QueryTotalSupplyRequest{},
		)
		if err != nil {
			sublogger.Error().Err(err).Msg("Could not get bank total supplyy")
			return
		}

		sublogger.Debug().
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying bank total supplyy")

		for _, coin := range response.Supply {
			if value, err := strconv.ParseFloat(coin.Amount.String(), 64); err != nil {
				sublogger.Error().
					Err(err).
					Msg("Could not get total supply")
			} else {
				generalSupplyTotalGauge.With(prometheus.Labels{
					"denom": coin.Denom,
				}).Set(value)
			}
		}
	}()
	wg.Add(1)

	wg.Wait()

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
	sublogger.Info().
		Str("method", "GET").
		Str("endpoint", "/metrics/general").
		Float64("request-time", time.Since(requestStart).Seconds()).
		Msg("Request processed")
}

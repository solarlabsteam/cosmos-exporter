package main

import (
	"context"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func ValidatorsHandler(w http.ResponseWriter, r *http.Request, grpcConn *grpc.ClientConn) {
	sublogger := log.With().
		Str("request-id", uuid.New().String()).
		Logger()

	validatorsCommissionGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_validators_commission",
			Help: "Commission of the Cosmos-based blockchain validator",
		},
		[]string{"address", "moniker", "denom"},
	)

	validatorsStatusGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_validators_status",
			Help: "Status of the Cosmos-based blockchain validator",
		},
		[]string{"address", "moniker"},
	)

	validatorsJailedGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_validators_jailed",
			Help: "Jailed status of the Cosmos-based blockchain validator",
		},
		[]string{"address", "moniker"},
	)

	validatorsTokensGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_validators_tokens",
			Help: "Tokens of the Cosmos-based blockchain validator",
		},
		[]string{"address", "moniker"},
	)

	validatorsDelegatorSharesGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_validators_delegator_shares",
			Help: "Delegator shares of the Cosmos-based blockchain validator",
		},
		[]string{"address", "moniker"},
	)

	validatorsMinSelfDelegationGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_validators_min_self_delegation",
			Help: "Self declared minimum self delegation shares of the Cosmos-based blockchain validator",
		},
		[]string{"address", "moniker"},
	)

	registry := prometheus.NewRegistry()
	registry.MustRegister(validatorsCommissionGauge)
	registry.MustRegister(validatorsStatusGauge)
	registry.MustRegister(validatorsJailedGauge)
	registry.MustRegister(validatorsTokensGauge)
	registry.MustRegister(validatorsDelegatorSharesGauge)
	registry.MustRegister(validatorsMinSelfDelegationGauge)

	stakingClient := stakingtypes.NewQueryClient(grpcConn)
	validators, err := stakingClient.Validators(
		context.Background(),
		&stakingtypes.QueryValidatorsRequest{},
	)
	if err != nil {
		sublogger.Error().Err(err).Msg("Could not get validators")
		return
	}

	for _, validator := range validators.Validators {
		// because cosmos's dec doesn't have .toFloat64() method or whatever and returns everything as int
		rate, err := strconv.ParseFloat(validator.Commission.CommissionRates.Rate.String(), 64)
		if err != nil {
			log.Error().
				Err(err).
				Str("address", validator.OperatorAddress).
				Msg("Could not get commission rate")
		} else {
			validatorsCommissionGauge.With(prometheus.Labels{
				"address": validator.OperatorAddress,
				"moniker": validator.Description.Moniker,
				"denom":   *Denom,
			}).Set(rate)
		}

		validatorsStatusGauge.With(prometheus.Labels{
			"address": validator.OperatorAddress,
			"moniker": validator.Description.Moniker,
		}).Set(float64(validator.Status))

		// golang doesn't have a ternary operator, so we have to stick with this ugly solution
		var jailed float64

		if validator.Jailed {
			jailed = 1
		} else {
			jailed = 0
		}
		validatorsJailedGauge.With(prometheus.Labels{
			"address": validator.OperatorAddress,
			"moniker": validator.Description.Moniker,
		}).Set(float64(jailed))

		validatorsTokensGauge.With(prometheus.Labels{
			"address": validator.OperatorAddress,
			"moniker": validator.Description.Moniker,
		}).Set(float64(validator.Tokens.Int64()))

		validatorsDelegatorSharesGauge.With(prometheus.Labels{
			"address": validator.OperatorAddress,
			"moniker": validator.Description.Moniker,
		}).Set(float64(validator.DelegatorShares.RoundInt64()))

		validatorsMinSelfDelegationGauge.With(prometheus.Labels{
			"address": validator.OperatorAddress,
			"moniker": validator.Description.Moniker,
		}).Set(float64(validator.MinSelfDelegation.Int64()))
	}

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
	sublogger.Info().
		Str("method", "GET").
		Str("endpoint", "/metrics/validators").
		Msg("Request processed")
}

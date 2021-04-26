package main

import (
	"context"
	"net/http"
	"strconv"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"

	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func ValidatorHandler(w http.ResponseWriter, r *http.Request, grpcConn *grpc.ClientConn) {
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

	validatorDelegationsGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_validator_delegations",
			Help: "Delegations of the Cosmos-based blockchain validator",
		},
		[]string{"address", "moniker", "denom", "delegated_by"},
	)

	validatorTokensGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_validator_tokens",
			Help: "Tokens of the Cosmos-based blockchain validator",
		},
		[]string{"address", "moniker"},
	)

	validatorDelegatorSharesGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_validator_delegators_shares",
			Help: "Delegators shares of the Cosmos-based blockchain validator",
		},
		[]string{"address", "moniker"},
	)

	validatorCommissionRateGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_validator_commission_rate",
			Help: "Commission rate of the Cosmos-based blockchain validator",
		},
		[]string{"address", "moniker"},
	)
	validatorCommissionGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_validator_commission",
			Help: "Commission of the Cosmos-based blockchain validator",
		},
		[]string{"address", "moniker", "denom"},
	)

	validatorUnbondingsGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_validator_unbondings",
			Help: "Unbondings of the Cosmos-based blockchain validator",
		},
		[]string{"address", "moniker", "denom", "unbonded_by"},
	)

	validatorRedelegationsGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_validator_redelegations",
			Help: "Redelegations of the Cosmos-based blockchain validator",
		},
		[]string{"address", "moniker", "denom", "redelegated_by", "redelegated_to"},
	)

	registry := prometheus.NewRegistry()
	registry.MustRegister(validatorDelegationsGauge)
	registry.MustRegister(validatorTokensGauge)
	registry.MustRegister(validatorDelegatorSharesGauge)
	registry.MustRegister(validatorCommissionRateGauge)
	registry.MustRegister(validatorCommissionGauge)
	registry.MustRegister(validatorUnbondingsGauge)
	registry.MustRegister(validatorRedelegationsGauge)

	// doing this not in goroutine as we'll need the moniker value later
	sublogger.Debug().
		Str("address", address).
		Msg("Started querying validator")

	stakingClient := stakingtypes.NewQueryClient(grpcConn)
	validator, err := stakingClient.Validator(
		context.Background(),
		&stakingtypes.QueryValidatorRequest{ValidatorAddr: myAddress.String()},
	)
	if err != nil {
		sublogger.Error().
			Str("address", address).
			Err(err).
			Msg("Could not get validator")
		return
	}

	sublogger.Debug().
		Str("address", address).
		Msg("Finished querying validator")

	validatorTokensGauge.With(prometheus.Labels{
		"address": validator.Validator.OperatorAddress,
		"moniker": validator.Validator.Description.Moniker,
	}).Set(float64(validator.Validator.Tokens.Int64()))

	validatorDelegatorSharesGauge.With(prometheus.Labels{
		"address": validator.Validator.OperatorAddress,
		"moniker": validator.Validator.Description.Moniker,
	}).Set(float64(validator.Validator.DelegatorShares.RoundInt64()))

	// because cosmos's dec doesn't have .toFloat64() method or whatever and returns everything as int
	rate, err := strconv.ParseFloat(validator.Validator.Commission.CommissionRates.Rate.String(), 64)
	if err != nil {
		sublogger.Error().
			Str("address", address).
			Err(err).
			Msg("Could not get commission rate")
	} else {
		validatorCommissionRateGauge.With(prometheus.Labels{
			"address": validator.Validator.OperatorAddress,
			"moniker": validator.Validator.Description.Moniker,
		}).Set(rate)
	}

	var wg sync.WaitGroup

	go func() {
		defer wg.Done()

		sublogger.Debug().
			Str("address", address).
			Msg("Started querying validator delegations")

		stakingClient := stakingtypes.NewQueryClient(grpcConn)
		stakingRes, err := stakingClient.ValidatorDelegations(
			context.Background(),
			&stakingtypes.QueryValidatorDelegationsRequest{ValidatorAddr: myAddress.String()},
		)
		if err != nil {
			sublogger.Error().
				Str("address", address).
				Err(err).
				Msg("Could not get validator delegations")
			return
		}

		sublogger.Debug().
			Str("address", address).
			Msg("Finished querying validator delegations")

		for _, delegation := range stakingRes.DelegationResponses {
			validatorDelegationsGauge.With(prometheus.Labels{
				"moniker":      validator.Validator.Description.Moniker,
				"address":      delegation.Delegation.ValidatorAddress,
				"denom":        delegation.Balance.Denom,
				"delegated_by": delegation.Delegation.DelegatorAddress,
			}).Set(float64(delegation.Balance.Amount.Int64()))
		}
	}()
	wg.Add(1)

	go func() {
		defer wg.Done()

		sublogger.Debug().
			Str("address", address).
			Msg("Started querying validator commission")

		distributionClient := distributiontypes.NewQueryClient(grpcConn)
		distributionRes, err := distributionClient.ValidatorCommission(
			context.Background(),
			&distributiontypes.QueryValidatorCommissionRequest{ValidatorAddress: myAddress.String()},
		)
		if err != nil {
			sublogger.Error().
				Str("address", address).
				Err(err).
				Msg("Could not get validator commission")
			return
		}

		sublogger.Debug().
			Str("address", address).
			Msg("Finished querying validator commission")

		for _, commission := range distributionRes.Commission.Commission {
			validatorCommissionGauge.With(prometheus.Labels{
				"address": address,
				"moniker": validator.Validator.Description.Moniker,
				"denom":   commission.Denom,
			}).Set(float64(commission.Amount.RoundInt64()))
		}
	}()
	wg.Add(1)

	go func() {
		defer wg.Done()

		sublogger.Debug().
			Str("address", address).
			Msg("Started querying validator unbonding delegations")

		stakingClient := stakingtypes.NewQueryClient(grpcConn)
		stakingRes, err := stakingClient.ValidatorUnbondingDelegations(
			context.Background(),
			&stakingtypes.QueryValidatorUnbondingDelegationsRequest{ValidatorAddr: myAddress.String()},
		)
		if err != nil {
			sublogger.Error().
				Str("address", address).
				Err(err).
				Msg("Could not get validator unbonding delegations")
			return
		}

		sublogger.Debug().
			Str("address", address).
			Msg("Finished querying validator unbonding delegations")

		for _, unbonding := range stakingRes.UnbondingResponses {
			var sum float64 = 0
			for _, entry := range unbonding.Entries {
				sum += float64(entry.Balance.Int64())
			}

			validatorUnbondingsGauge.With(prometheus.Labels{
				"address":     unbonding.ValidatorAddress,
				"moniker":     validator.Validator.Description.Moniker,
				"denom":       *Denom, // unbonding does not have denom in response for some reason
				"unbonded_by": unbonding.DelegatorAddress,
			}).Set(sum)
		}
	}()
	wg.Add(1)

	go func() {
		defer wg.Done()

		sublogger.Debug().
			Str("address", address).
			Msg("Started querying validator redelegations")

		stakingClient := stakingtypes.NewQueryClient(grpcConn)
		stakingRes, err := stakingClient.Redelegations(
			context.Background(),
			&stakingtypes.QueryRedelegationsRequest{SrcValidatorAddr: myAddress.String()},
		)
		if err != nil {
			sublogger.Error().
				Str("address", address).
				Err(err).
				Msg("Could not get redelegations")
			return
		}

		sublogger.Debug().
			Str("address", address).
			Msg("Finished querying validator redelegations")

		for _, redelegation := range stakingRes.RedelegationResponses {
			var sum float64 = 0
			for _, entry := range redelegation.Entries {
				sum += float64(entry.Balance.Int64())
			}

			validatorRedelegationsGauge.With(prometheus.Labels{
				"address":        redelegation.Redelegation.ValidatorSrcAddress,
				"moniker":        validator.Validator.Description.Moniker,
				"denom":          *Denom, // redelegation does not have denom in response for some reason
				"redelegated_by": redelegation.Redelegation.DelegatorAddress,
				"redelegated_to": redelegation.Redelegation.ValidatorDstAddress,
			}).Set(sum)
		}
	}()
	wg.Add(1)

	wg.Wait()

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
	sublogger.Info().
		Str("method", "GET").
		Str("endpoint", "/metrics/validator?address="+address).
		Msg("Request processed")
}

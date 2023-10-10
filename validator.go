package main

import (
	"context"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	crytpocode "github.com/cosmos/cosmos-sdk/crypto/codec"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/rs/zerolog"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	querytypes "github.com/cosmos/cosmos-sdk/types/query"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type ValidatorMetrics struct {
	tokensGauge          *prometheus.GaugeVec
	delegatorSharesGauge *prometheus.GaugeVec
	commissionRateGauge  *prometheus.GaugeVec
	statusGauge          *prometheus.GaugeVec
	jailedGauge          *prometheus.GaugeVec
	missedBlocksGauge    *prometheus.GaugeVec
}
type ValidatorExtendedMetrics struct {
	delegationsGauge   *prometheus.GaugeVec
	commissionGauge    *prometheus.GaugeVec
	rewardsGauge       *prometheus.GaugeVec
	unbondingsGauge    *prometheus.GaugeVec
	redelegationsGauge *prometheus.GaugeVec

	rankGauge     *prometheus.GaugeVec
	isActiveGauge *prometheus.GaugeVec
}

func NewValidatorMetrics(reg prometheus.Registerer) *ValidatorMetrics {
	m := &ValidatorMetrics{

		tokensGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "cosmos_validator_tokens",
				Help:        "Tokens of the Cosmos-based blockchain validator",
				ConstLabels: ConstLabels,
			},
			[]string{"address", "moniker", "denom"},
		),

		delegatorSharesGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "cosmos_validator_delegators_shares",
				Help:        "Delegators shares of the Cosmos-based blockchain validator",
				ConstLabels: ConstLabels,
			},
			[]string{"address", "moniker", "denom"},
		),

		commissionRateGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "cosmos_validator_commission_rate",
				Help:        "Commission rate of the Cosmos-based blockchain validator",
				ConstLabels: ConstLabels,
			},
			[]string{"address", "moniker"},
		),

		statusGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "cosmos_validator_status",
				Help:        "Status of the Cosmos-based blockchain validator",
				ConstLabels: ConstLabels,
			},
			[]string{"address", "moniker"},
		),

		jailedGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "cosmos_validator_jailed",
				Help:        "1 if the Cosmos-based blockchain validator is jailed, 0 if no",
				ConstLabels: ConstLabels,
			},
			[]string{"address", "moniker"},
		),
		missedBlocksGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "cosmos_validator_missed_blocks",
				Help:        "Missed blocks of the Cosmos-based blockchain validator",
				ConstLabels: ConstLabels,
			},
			[]string{"address", "moniker"},
		),
	}

	reg.MustRegister(m.tokensGauge)
	reg.MustRegister(m.delegatorSharesGauge)
	reg.MustRegister(m.commissionRateGauge)
	reg.MustRegister(m.statusGauge)
	reg.MustRegister(m.jailedGauge)
	reg.MustRegister(m.missedBlocksGauge)

	return m
}

func NewValidatorExtendedMetrics(reg prometheus.Registerer) *ValidatorExtendedMetrics {
	m := &ValidatorExtendedMetrics{

		delegationsGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "cosmos_validator_delegations",
				Help:        "Delegations of the Cosmos-based blockchain validator",
				ConstLabels: ConstLabels,
			},
			[]string{"address", "moniker", "denom", "delegated_by"},
		),

		commissionGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "cosmos_validator_commission",
				Help:        "Commission of the Cosmos-based blockchain validator",
				ConstLabels: ConstLabels,
			},
			[]string{"address", "moniker", "denom"},
		),
		rewardsGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "cosmos_validator_rewards",
				Help:        "Rewards of the Cosmos-based blockchain validator",
				ConstLabels: ConstLabels,
			},
			[]string{"address", "moniker", "denom"},
		),

		unbondingsGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "cosmos_validator_unbondings",
				Help:        "Unbondings of the Cosmos-based blockchain validator",
				ConstLabels: ConstLabels,
			},
			[]string{"address", "moniker", "denom", "unbonded_by"},
		),

		redelegationsGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "cosmos_validator_redelegations",
				Help:        "Redelegations of the Cosmos-based blockchain validator",
				ConstLabels: ConstLabels,
			},
			[]string{"address", "moniker", "denom", "redelegated_by", "redelegated_to"},
		),

		rankGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "cosmos_validator_rank",
				Help:        "Rank of the Cosmos-based blockchain validator",
				ConstLabels: ConstLabels,
			},
			[]string{"address", "moniker"},
		),

		isActiveGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "cosmos_validator_active",
				Help:        "1 if the Cosmos-based blockchain validator is in active set, 0 if no",
				ConstLabels: ConstLabels,
			},
			[]string{"address", "moniker"},
		),
	}

	reg.MustRegister(m.delegationsGauge)

	reg.MustRegister(m.commissionGauge)
	reg.MustRegister(m.rewardsGauge)
	reg.MustRegister(m.unbondingsGauge)
	reg.MustRegister(m.redelegationsGauge)

	reg.MustRegister(m.rankGauge)
	reg.MustRegister(m.isActiveGauge)

	return m
}
func getValidatorBasicMetrics(wg *sync.WaitGroup, sublogger *zerolog.Logger, metrics *ValidatorMetrics, s *service, validatorAddress sdk.ValAddress) *stakingtypes.QueryValidatorResponse {

	// doing this not in goroutine as we'll need the moniker value later
	sublogger.Debug().
		Str("address", validatorAddress.String()).
		Msg("Started querying validator")
	validatorQueryStart := time.Now()

	stakingClient := stakingtypes.NewQueryClient(s.grpcConn)
	validator, err := stakingClient.Validator(
		context.Background(),
		&stakingtypes.QueryValidatorRequest{ValidatorAddr: validatorAddress.String()},
	)
	if err != nil {
		sublogger.Error().
			Str("address", validatorAddress.String()).
			Err(err).
			Msg("Could not get validator")
		return nil
	}

	sublogger.Debug().
		Str("address", validatorAddress.String()).
		Float64("request-time", time.Since(validatorQueryStart).Seconds()).
		Msg("Finished querying validator")

	if value, err := strconv.ParseFloat(validator.Validator.Tokens.String(), 64); err != nil {
		sublogger.Error().
			Str("address", validatorAddress.String()).
			Err(err).
			Msg("Could not parse validator tokens")
	} else {
		metrics.tokensGauge.With(prometheus.Labels{
			"address": validator.Validator.OperatorAddress,
			"moniker": validator.Validator.Description.Moniker,
			"denom":   Denom,
		}).Set(value / DenomCoefficient)
	}

	// because cosmos's dec doesn't have .toFloat64() method or whatever and returns everything as int
	if value, err := strconv.ParseFloat(validator.Validator.DelegatorShares.String(), 64); err != nil {
		sublogger.Error().
			Str("address", validatorAddress.String()).
			Err(err).
			Msg("Could not parse delegator shares")
	} else {
		metrics.delegatorSharesGauge.With(prometheus.Labels{
			"address": validator.Validator.OperatorAddress,
			"moniker": validator.Validator.Description.Moniker,
			"denom":   Denom,
		}).Set(value / DenomCoefficient)
	}

	// because cosmos's dec doesn't have .toFloat64() method or whatever and returns everything as int
	if rate, err := strconv.ParseFloat(validator.Validator.Commission.CommissionRates.Rate.String(), 64); err != nil {
		sublogger.Error().
			Str("address", validatorAddress.String()).
			Err(err).
			Msg("Could not parse commission rate")
	} else {
		metrics.commissionRateGauge.With(prometheus.Labels{
			"address": validator.Validator.OperatorAddress,
			"moniker": validator.Validator.Description.Moniker,
		}).Set(rate)
	}

	metrics.statusGauge.With(prometheus.Labels{
		"address": validator.Validator.OperatorAddress,
		"moniker": validator.Validator.Description.Moniker,
	}).Set(float64(validator.Validator.Status))

	// golang doesn't have a ternary operator, so we have to stick with this ugly solution
	var jailed float64

	if validator.Validator.Jailed {
		jailed = 1
	} else {
		jailed = 0
	}
	metrics.jailedGauge.With(prometheus.Labels{
		"address": validator.Validator.OperatorAddress,
		"moniker": validator.Validator.Description.Moniker,
	}).Set(jailed)

	wg.Add(1)
	go func() {
		defer wg.Done()

		sublogger.Debug().
			Str("address", validatorAddress.String()).
			Msg("Started querying validator signing info")
		queryStart := time.Now()
		interfaceRegistry := codectypes.NewInterfaceRegistry()

		crytpocode.RegisterInterfaces(interfaceRegistry)

		err := validator.Validator.UnpackInterfaces(interfaceRegistry) // Unpack interfaces, to populate the Anys' cached values
		if err != nil {
			sublogger.Error().
				Str("address", validatorAddress.String()).
				Err(err).
				Msg("Could not get unpack validator inferfaces")
		}

		pubKey, err := validator.Validator.GetConsAddr()
		if err != nil {
			sublogger.Error().
				Str("address", validatorAddress.String()).
				Err(err).
				Msg("Could not get validator pubkey")
		}

		slashingClient := slashingtypes.NewQueryClient(s.grpcConn)
		slashingRes, err := slashingClient.SigningInfo(
			context.Background(),
			&slashingtypes.QuerySigningInfoRequest{ConsAddress: pubKey.String()},
		)
		if err != nil {
			sublogger.Error().
				Str("address", validatorAddress.String()).
				Err(err).
				Msg("Could not get validator signing info")
			return
		}

		sublogger.Debug().
			Str("address", validatorAddress.String()).
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying validator signing info")

		sublogger.Debug().
			Str("address", validatorAddress.String()).
			Int64("missedBlocks", slashingRes.ValSigningInfo.MissedBlocksCounter).
			Msg("Finished querying validator signing info")

		metrics.missedBlocksGauge.With(prometheus.Labels{
			"moniker": validator.Validator.Description.Moniker,
			"address": validatorAddress.String(),
		}).Set(float64(slashingRes.ValSigningInfo.MissedBlocksCounter))
	}()

	return validator
}
func getValidatorExtendedMetrics(wg *sync.WaitGroup, sublogger *zerolog.Logger, metrics *ValidatorExtendedMetrics, s *service, validatorAddress sdk.ValAddress, moniker string, validator *stakingtypes.QueryValidatorResponse) {

	wg.Add(1)
	go func() {
		defer wg.Done()

		sublogger.Debug().
			Str("address", validatorAddress.String()).
			Msg("Started querying validator delegations")
		queryStart := time.Now()

		stakingClient := stakingtypes.NewQueryClient(s.grpcConn)
		stakingRes, err := stakingClient.ValidatorDelegations(
			context.Background(),
			&stakingtypes.QueryValidatorDelegationsRequest{
				ValidatorAddr: validatorAddress.String(),
				Pagination: &querytypes.PageRequest{
					Limit: Limit,
				},
			},
		)
		if err != nil {
			sublogger.Error().
				Str("address", validatorAddress.String()).
				Err(err).
				Msg("Could not get validator delegations")
			return
		}

		sublogger.Debug().
			Str("address", validatorAddress.String()).
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying validator delegations")

		for _, delegation := range stakingRes.DelegationResponses {
			value, err := strconv.ParseFloat(delegation.Balance.Amount.String(), 64)
			if err != nil {
				log.Error().
					Err(err).
					Str("address", validatorAddress.String()).
					Msg("Could not convert delegation entry")
			} else {
				metrics.delegationsGauge.With(prometheus.Labels{
					"moniker":      moniker,
					"address":      delegation.Delegation.ValidatorAddress,
					"denom":        Denom,
					"delegated_by": delegation.Delegation.DelegatorAddress,
				}).Set(value / DenomCoefficient)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		sublogger.Debug().
			Str("address", validatorAddress.String()).
			Msg("Started querying validator commission")
		queryStart := time.Now()

		distributionClient := distributiontypes.NewQueryClient(s.grpcConn)
		distributionRes, err := distributionClient.ValidatorCommission(
			context.Background(),
			&distributiontypes.QueryValidatorCommissionRequest{ValidatorAddress: validatorAddress.String()},
		)
		if err != nil {
			sublogger.Error().
				Str("address", validatorAddress.String()).
				Err(err).
				Msg("Could not get validator commission")
			return
		}

		sublogger.Debug().
			Str("address", validatorAddress.String()).
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying validator commission")

		for _, commission := range distributionRes.Commission.Commission {
			// because cosmos's dec doesn't have .toFloat64() method or whatever and returns everything as int
			value, err := strconv.ParseFloat(commission.Amount.String(), 64)
			if err != nil {
				log.Error().
					Err(err).
					Str("address", validatorAddress.String()).
					Msg("Could not get validator commission")
			} else {
				metrics.commissionGauge.With(prometheus.Labels{
					"address": validatorAddress.String(),
					"moniker": moniker,
					"denom":   Denom,
				}).Set(value / DenomCoefficient)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		sublogger.Debug().
			Str("address", validatorAddress.String()).
			Msg("Started querying validator rewards")
		queryStart := time.Now()

		distributionClient := distributiontypes.NewQueryClient(s.grpcConn)
		distributionRes, err := distributionClient.ValidatorOutstandingRewards(
			context.Background(),
			&distributiontypes.QueryValidatorOutstandingRewardsRequest{ValidatorAddress: validatorAddress.String()},
		)
		if err != nil {
			sublogger.Error().
				Str("address", validatorAddress.String()).
				Err(err).
				Msg("Could not get validator rewards")
			return
		}

		sublogger.Debug().
			Str("address", validatorAddress.String()).
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying validator rewards")

		for _, reward := range distributionRes.Rewards.Rewards {
			// because cosmos's dec doesn't have .toFloat64() method or whatever and returns everything as int
			if value, err := strconv.ParseFloat(reward.Amount.String(), 64); err != nil {
				sublogger.Error().
					Str("address", validatorAddress.String()).
					Err(err).
					Msg("Could not get reward")
			} else {
				metrics.rewardsGauge.With(prometheus.Labels{
					"address": validatorAddress.String(),
					"moniker": moniker,
					"denom":   Denom,
				}).Set(value / DenomCoefficient)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		sublogger.Debug().
			Str("address", validatorAddress.String()).
			Msg("Started querying validator unbonding delegations")
		queryStart := time.Now()

		stakingClient := stakingtypes.NewQueryClient(s.grpcConn)
		stakingRes, err := stakingClient.ValidatorUnbondingDelegations(
			context.Background(),
			&stakingtypes.QueryValidatorUnbondingDelegationsRequest{ValidatorAddr: validatorAddress.String()},
		)
		if err != nil {
			sublogger.Error().
				Str("address", validatorAddress.String()).
				Err(err).
				Msg("Could not get validator unbonding delegations")
			return
		}

		sublogger.Debug().
			Str("address", validatorAddress.String()).
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying validator unbonding delegations")

		for _, unbonding := range stakingRes.UnbondingResponses {
			var sum float64 = 0
			for _, entry := range unbonding.Entries {
				value, err := strconv.ParseFloat(entry.Balance.String(), 64)
				if err != nil {
					log.Error().
						Err(err).
						Str("address", validatorAddress.String()).
						Msg("Could not convert unbonding delegation entry")
				} else {
					sum += value
				}
			}

			metrics.unbondingsGauge.With(prometheus.Labels{
				"address":     unbonding.ValidatorAddress,
				"moniker":     moniker,
				"denom":       Denom, // unbonding does not have denom in response for some reason
				"unbonded_by": unbonding.DelegatorAddress,
			}).Set(sum / DenomCoefficient)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		sublogger.Debug().
			Str("address", validatorAddress.String()).
			Msg("Started querying validator redelegations")
		queryStart := time.Now()

		stakingClient := stakingtypes.NewQueryClient(s.grpcConn)
		stakingRes, err := stakingClient.Redelegations(
			context.Background(),
			&stakingtypes.QueryRedelegationsRequest{SrcValidatorAddr: validatorAddress.String()},
		)
		if err != nil {
			sublogger.Error().
				Str("address", validatorAddress.String()).
				Err(err).
				Msg("Could not get redelegations")
			return
		}

		sublogger.Debug().
			Str("address", validatorAddress.String()).
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying validator redelegations")

		for _, redelegation := range stakingRes.RedelegationResponses {
			var sum float64 = 0
			for _, entry := range redelegation.Entries {
				value, err := strconv.ParseFloat(entry.Balance.String(), 64)
				if err != nil {
					log.Error().
						Err(err).
						Str("address", validatorAddress.String()).
						Msg("Could not convert redelegation entry")
				} else {
					sum += value
				}
			}

			metrics.redelegationsGauge.With(prometheus.Labels{
				"address":        redelegation.Redelegation.ValidatorSrcAddress,
				"moniker":        moniker,
				"denom":          Denom, // redelegation does not have denom in response for some reason
				"redelegated_by": redelegation.Redelegation.DelegatorAddress,
				"redelegated_to": redelegation.Redelegation.ValidatorDstAddress,
			}).Set(sum / DenomCoefficient)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		sublogger.Debug().
			Str("address", validatorAddress.String()).
			Msg("Started querying validator other validators")
		queryStart := time.Now()

		stakingClient := stakingtypes.NewQueryClient(s.grpcConn)
		stakingRes, err := stakingClient.Validators(
			context.Background(),
			&stakingtypes.QueryValidatorsRequest{
				Pagination: &querytypes.PageRequest{
					Limit: Limit,
				},
			},
		)
		if err != nil {
			sublogger.Error().
				Str("address", validatorAddress.String()).
				Err(err).
				Msg("Could not get other validators")
			return
		}

		sublogger.Debug().
			Str("address", validatorAddress.String()).
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying validator other validators")

		validators := stakingRes.Validators

		// sorting by delegator shares to display rankings (unbonded go last)
		sort.Slice(validators, func(i, j int) bool {
			firstShares, firstErr := strconv.ParseFloat(validators[i].DelegatorShares.String(), 64)
			secondShares, secondErr := strconv.ParseFloat(validators[j].DelegatorShares.String(), 64)
			if !validators[i].IsBonded() && validators[j].IsBonded() {
				return false
			} else if validators[i].IsBonded() && !validators[j].IsBonded() {
				return true
			}

			if firstErr != nil || secondErr != nil {
				sublogger.Error().
					Err(err).
					Msg("Error converting delegator shares for sorting")
				return true
			}

			return firstShares > secondShares
		})

		var validatorRank int

		for index, validatorIterated := range validators {
			if validatorIterated.OperatorAddress == validator.Validator.OperatorAddress {
				validatorRank = index + 1
				break
			}
		}

		if validatorRank == 0 {
			sublogger.Warn().
				Str("address", validatorAddress.String()).
				Msg("Could not find validator in validators list")
			return
		}

		metrics.rankGauge.With(prometheus.Labels{
			"moniker": moniker,
			"address": validatorAddress.String(),
		}).Set(float64(validatorRank))

		sublogger.Debug().
			Str("address", validatorAddress.String()).
			Msg("Started querying validator params")
		queryStart = time.Now()

		paramsRes, err := stakingClient.Params(
			context.Background(),
			&stakingtypes.QueryParamsRequest{},
		)
		if err != nil {
			sublogger.Error().
				Str("address", validatorAddress.String()).
				Err(err).
				Msg("Could not get params")
			return
		}

		sublogger.Debug().
			Str("address", validatorAddress.String()).
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying validator params")

		// golang doesn't have a ternary operator, so we have to stick with this ugly solution
		var active float64

		if validatorRank <= int(paramsRes.Params.MaxValidators) {
			active = 1
		} else {
			active = 0
		}

		metrics.isActiveGauge.With(prometheus.Labels{
			"address": validator.Validator.OperatorAddress,
			"moniker": moniker,
		}).Set(active)
	}()

}
func (s *service) ValidatorHandler(w http.ResponseWriter, r *http.Request) {
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
	validatorMetrics := NewValidatorMetrics(registry)
	validatorExtendedMetrics := NewValidatorExtendedMetrics(registry)
	var wg sync.WaitGroup

	validator := getValidatorBasicMetrics(&wg, &sublogger, validatorMetrics, s, myAddress)
	if validator != nil {
		getValidatorExtendedMetrics(&wg, &sublogger, validatorExtendedMetrics, s, myAddress, validator.Validator.Description.Moniker, validator)
	}

	wg.Wait()

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
	sublogger.Info().
		Str("method", "GET").
		Str("endpoint", "/metrics/validator?address="+address).
		Float64("request-time", time.Since(requestStart).Seconds()).
		Msg("Request processed")
}

package main

import (
	"context"
	"github.com/rs/zerolog"
	"net/http"
	"strconv"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type WalletMetrics struct {
	balanceGauge *prometheus.GaugeVec
}
type WalletExtendedMetrics struct {
	delegationGauge   *prometheus.GaugeVec
	redelegationGauge *prometheus.GaugeVec
	unbondingsGauge   *prometheus.GaugeVec
	rewardsGauge      *prometheus.GaugeVec
}

func NewWalletMetrics(reg prometheus.Registerer) *WalletMetrics {
	m := &WalletMetrics{

		balanceGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "cosmos_wallet_balance",
				Help:        "Balance of the Cosmos-based blockchain wallet",
				ConstLabels: ConstLabels,
			},
			[]string{"address", "denom"},
		),
	}
	reg.MustRegister(m.balanceGauge)

	return m
}
func NewWalletExtendedMetrics(reg prometheus.Registerer) *WalletExtendedMetrics {
	m := &WalletExtendedMetrics{
		delegationGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "cosmos_wallet_delegations",
				Help:        "Delegations of the Cosmos-based blockchain wallet",
				ConstLabels: ConstLabels,
			},
			[]string{"address", "denom", "delegated_to"},
		),

		redelegationGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "cosmos_wallet_redelegations",
				Help:        "Redlegations of the Cosmos-based blockchain wallet",
				ConstLabels: ConstLabels,
			},
			[]string{"address", "denom", "redelegated_from", "redelegated_to"},
		),

		unbondingsGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "cosmos_wallet_unbondings",
				Help:        "Unbondings of the Cosmos-based blockchain wallet",
				ConstLabels: ConstLabels,
			},
			[]string{"address", "denom", "unbonded_from"},
		),

		rewardsGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "cosmos_wallet_rewards",
				Help:        "Rewards of the Cosmos-based blockchain wallet",
				ConstLabels: ConstLabels,
			},
			[]string{"address", "denom", "validator_address"},
		),
	}

	reg.MustRegister(m.delegationGauge)
	reg.MustRegister(m.unbondingsGauge)
	reg.MustRegister(m.redelegationGauge)
	reg.MustRegister(m.rewardsGauge)

	return m
}
func getWalletMetrics(wg *sync.WaitGroup, sublogger *zerolog.Logger, metrics *WalletMetrics, s *service, address sdk.AccAddress, allBalances bool) {

	wg.Add(1)
	go func() {
		defer wg.Done()
		sublogger.Debug().
			Str("address", address.String()).
			Msg("Started querying balance")
		queryStart := time.Now()

		bankClient := banktypes.NewQueryClient(s.grpcConn)

		if allBalances {
			bankRes, err := bankClient.AllBalances(
				context.Background(),
				&banktypes.QueryAllBalancesRequest{Address: address.String()},
			)

			if err != nil {
				sublogger.Error().
					Str("address", address.String()).
					Err(err).
					Msg("Could not get balance")
				return
			}

			sublogger.Debug().
				Str("address", address.String()).
				Float64("request-time", time.Since(queryStart).Seconds()).
				Msg("Finished querying all balances")

			for _, balance := range bankRes.Balances {

				// because cosmos dec doesn't have .toFloat64() method or whatever and returns everything as int
				if value, err := strconv.ParseFloat(balance.Amount.String(), 64); err != nil {
					sublogger.Error().
						Str("address", address.String()).
						Err(err).
						Msg("Could not parse balance")
				} else {
					metrics.balanceGauge.With(prometheus.Labels{
						"address": address.String(),
						"denom":   balance.Denom,
					}).Set(value / DenomCoefficient)
				}
			}
		} else {
			bankRes, err := bankClient.Balance(
				context.Background(),
				&banktypes.QueryBalanceRequest{Address: address.String(), Denom: Denom},
			)

			if err != nil {
				sublogger.Error().
					Str("address", address.String()).
					Err(err).
					Msg("Could not get balance")
				return
			}

			sublogger.Debug().
				Str("address", address.String()).
				Str("denom", Denom).
				Float64("request-time", time.Since(queryStart).Seconds()).
				Msg("Finished querying balance")
			balance := bankRes.Balance

			// because cosmos dec doesn't have .toFloat64() method or whatever and returns everything as int
			if value, err := strconv.ParseFloat(balance.Amount.String(), 64); err != nil {
				sublogger.Error().
					Str("address", address.String()).
					Err(err).
					Msg("Could not parse balance")
			} else {
				metrics.balanceGauge.With(prometheus.Labels{
					"address": address.String(),
					"denom":   balance.Denom,
				}).Set(value / DenomCoefficient)
			}
		}

	}()

}
func getWalletExtendedMetrics(wg *sync.WaitGroup, sublogger *zerolog.Logger, metrics *WalletExtendedMetrics, s *service, address sdk.AccAddress) {

	wg.Add(1)
	go func() {
		defer wg.Done()
		sublogger.Debug().
			Str("address", address.String()).
			Msg("Started querying delegations")
		queryStart := time.Now()

		stakingClient := stakingtypes.NewQueryClient(s.grpcConn)
		stakingRes, err := stakingClient.DelegatorDelegations(
			context.Background(),
			&stakingtypes.QueryDelegatorDelegationsRequest{DelegatorAddr: address.String()},
		)
		if err != nil {
			sublogger.Error().
				Str("address", address.String()).
				Err(err).
				Msg("Could not get delegations")
			return
		}

		sublogger.Debug().
			Str("address", address.String()).
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying delegations")

		for _, delegation := range stakingRes.DelegationResponses {
			// because cosmos's dec doesn't have .toFloat64() method or whatever and returns everything as int
			if value, err := strconv.ParseFloat(delegation.Balance.Amount.String(), 64); err != nil {
				sublogger.Error().
					Str("address", address.String()).
					Err(err).
					Msg("Could not get delegation")
			} else {
				metrics.delegationGauge.With(prometheus.Labels{
					"address":      address.String(),
					"denom":        Denom,
					"delegated_to": delegation.Delegation.ValidatorAddress,
				}).Set(value / DenomCoefficient)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		sublogger.Debug().
			Str("address", address.String()).
			Msg("Started querying unbonding delegations")
		queryStart := time.Now()

		stakingClient := stakingtypes.NewQueryClient(s.grpcConn)
		stakingRes, err := stakingClient.DelegatorUnbondingDelegations(
			context.Background(),
			&stakingtypes.QueryDelegatorUnbondingDelegationsRequest{DelegatorAddr: address.String()},
		)
		if err != nil {
			sublogger.Error().
				Str("address", address.String()).
				Err(err).
				Msg("Could not get unbonding delegations")
			return
		}

		sublogger.Debug().
			Str("address", address.String()).
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying unbonding delegations")

		for _, unbonding := range stakingRes.UnbondingResponses {
			var sum float64 = 0
			for _, entry := range unbonding.Entries {
				// because cosmos's dec doesn't have .toFloat64() method or whatever and returns everything as int
				if value, err := strconv.ParseFloat(entry.Balance.String(), 64); err != nil {
					sublogger.Error().
						Str("address", address.String()).
						Err(err).
						Msg("Could not parse unbonding delegation")
				} else {
					sum += value
				}
			}

			metrics.unbondingsGauge.With(prometheus.Labels{
				"address":       unbonding.DelegatorAddress,
				"denom":         Denom, // unbonding does not have denom in response for some reason
				"unbonded_from": unbonding.ValidatorAddress,
			}).Set(sum / DenomCoefficient)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		sublogger.Debug().
			Str("address", address.String()).
			Msg("Started querying redelegations")
		queryStart := time.Now()

		stakingClient := stakingtypes.NewQueryClient(s.grpcConn)
		stakingRes, err := stakingClient.Redelegations(
			context.Background(),
			&stakingtypes.QueryRedelegationsRequest{DelegatorAddr: address.String()},
		)
		if err != nil {
			sublogger.Error().
				Str("address", address.String()).
				Err(err).
				Msg("Could not get redelegations")
			return
		}

		sublogger.Debug().
			Str("address", address.String()).
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying redelegations")

		for _, redelegation := range stakingRes.RedelegationResponses {
			var sum float64 = 0
			for _, entry := range redelegation.Entries {
				// because cosmos's dec doesn't have .toFloat64() method or whatever and returns everything as int
				if value, err := strconv.ParseFloat(entry.Balance.String(), 64); err != nil {
					sublogger.Error().
						Str("address", address.String()).
						Err(err).
						Msg("Could not parse redelegation")
				} else {
					sum += value
				}
			}

			metrics.redelegationGauge.With(prometheus.Labels{
				"address":          redelegation.Redelegation.DelegatorAddress,
				"denom":            Denom, // redelegation does not have denom in response for some reason
				"redelegated_from": redelegation.Redelegation.ValidatorSrcAddress,
				"redelegated_to":   redelegation.Redelegation.ValidatorDstAddress,
			}).Set(sum / DenomCoefficient)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		sublogger.Debug().
			Str("address", address.String()).
			Msg("Started querying rewards")
		queryStart := time.Now()

		distributionClient := distributiontypes.NewQueryClient(s.grpcConn)
		distributionRes, err := distributionClient.DelegationTotalRewards(
			context.Background(),
			&distributiontypes.QueryDelegationTotalRewardsRequest{DelegatorAddress: address.String()},
		)
		if err != nil {
			sublogger.Error().
				Str("address", address.String()).
				Err(err).
				Msg("Could not get rewards")
			return
		}
		sublogger.Debug().
			Str("address", address.String()).
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying rewards")

		for _, reward := range distributionRes.Rewards {
			for _, entry := range reward.Reward {
				// because cosmos's dec doesn't have .toFloat64() method or whatever and returns everything as int
				if value, err := strconv.ParseFloat(entry.Amount.String(), 64); err != nil {
					sublogger.Error().
						Str("address", address.String()).
						Err(err).
						Msg("Could not parse reward")
				} else {
					metrics.rewardsGauge.With(prometheus.Labels{
						"address":           address.String(),
						"denom":             Denom,
						"validator_address": reward.ValidatorAddress,
					}).Set(value / DenomCoefficient)
				}
			}
		}
	}()

}
func (s *service) WalletHandler(w http.ResponseWriter, r *http.Request) {
	requestStart := time.Now()

	sublogger := log.With().
		Str("request-id", uuid.New().String()).
		Logger()

	address := r.URL.Query().Get("address")
	myAddress, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		sublogger.Error().
			Str("address", address).
			Err(err).
			Msg("Could not get address")
		return
	}

	registry := prometheus.NewRegistry()
	walletMetrics := NewWalletMetrics(registry)
	walletExtendedMetrics := NewWalletExtendedMetrics(registry)

	var wg sync.WaitGroup
	getWalletMetrics(&wg, &sublogger, walletMetrics, s, myAddress, true)
	getWalletExtendedMetrics(&wg, &sublogger, walletExtendedMetrics, s, myAddress)
	wg.Wait()

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
	sublogger.Info().
		Str("method", "GET").
		Str("endpoint", "/metrics/wallet?address="+address).
		Float64("request-time", time.Since(requestStart).Seconds()).
		Msg("Request processed")
}

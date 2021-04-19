package main

import (
	"context"
	"flag"
	"net/http"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	log "github.com/sirupsen/logrus"
)

var prefix = flag.String("bech-prefix", "persistence", "Bech32 prefix for the network")
var denom = flag.String("denom", "uxprt", "Cosmos coin denom")
var listenAddress = flag.String("listen-address", ":9300", "The address this exporter would listen on")
var nodeAddress = flag.String("node", "localhost:9090", "RPC node address")

func walletHandler(w http.ResponseWriter, r *http.Request, grpcConn *grpc.ClientConn) {
	address := r.URL.Query().Get("address")
	myAddress, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		log.Error("Could not get address for \"", address, "\", got error: ", err)
		return
	}

	walletBalanceGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_wallet_balance",
			Help: "Balance of the Cosmos-based blockchain wallet",
		},
		[]string{"address", "denom"},
	)

	walletDelegationGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_wallet_delegations",
			Help: "Delegations of the Cosmos-based blockchain wallet",
		},
		[]string{"address", "denom", "delegated_to"},
	)

	walletRedelegationGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_wallet_redelegations",
			Help: "Redlegations of the Cosmos-based blockchain wallet",
		},
		[]string{"address", "denom", "redelegated_from", "redelegated_to"},
	)

	walletUnbondingsGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_wallet_unbondings",
			Help: "Unbondings of the Cosmos-based blockchain wallet",
		},
		[]string{"address", "denom", "unbonded_from"},
	)

	walletRewardsGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_wallet_rewards",
			Help: "Rewards of the Cosmos-based blockchain wallet",
		},
		[]string{"address", "denom", "validator_address"},
	)

	registry := prometheus.NewRegistry()
	registry.MustRegister(walletBalanceGauge)
	registry.MustRegister(walletDelegationGauge)
	registry.MustRegister(walletUnbondingsGauge)
	registry.MustRegister(walletRedelegationGauge)
	registry.MustRegister(walletRewardsGauge)

	var wg sync.WaitGroup

	go func() {
		defer wg.Done()

		bankClient := banktypes.NewQueryClient(grpcConn)
		bankRes, err := bankClient.Balance(
			context.Background(),
			&banktypes.QueryBalanceRequest{Address: myAddress.String(), Denom: *denom},
		)
		if err != nil {
			log.Error("Could not get balance for \"", address, "\", got error: ", err)
			return
		}

		walletBalanceGauge.With(prometheus.Labels{
			"address": address,
			"denom":   bankRes.GetBalance().Denom,
		}).Set(float64(bankRes.GetBalance().Amount.Int64()))
	}()
	wg.Add(1)

	go func() {
		defer wg.Done()

		stakingClient := stakingtypes.NewQueryClient(grpcConn)
		stakingRes, err := stakingClient.DelegatorDelegations(
			context.Background(),
			&stakingtypes.QueryDelegatorDelegationsRequest{DelegatorAddr: myAddress.String()},
		)
		if err != nil {
			log.Error("Could not get delegations for \"", address, "\", got error: ", err)
			return
		}

		for _, delegation := range stakingRes.DelegationResponses {
			walletDelegationGauge.With(prometheus.Labels{
				"address":      address,
				"denom":        delegation.Balance.Denom,
				"delegated_to": delegation.Delegation.ValidatorAddress,
			}).Set(float64(delegation.Balance.Amount.Int64()))
		}
	}()
	wg.Add(1)

	go func() {
		defer wg.Done()

		stakingClient := stakingtypes.NewQueryClient(grpcConn)
		stakingRes, err := stakingClient.DelegatorUnbondingDelegations(
			context.Background(),
			&stakingtypes.QueryDelegatorUnbondingDelegationsRequest{DelegatorAddr: myAddress.String()},
		)
		if err != nil {
			log.Error("Could not get unbonding delegations for \"", address, "\", got error: ", err)
			return
		}

		for _, unbonding := range stakingRes.UnbondingResponses {
			var sum float64 = 0
			for _, entry := range unbonding.Entries {
				sum += float64(entry.Balance.Int64())
			}

			walletUnbondingsGauge.With(prometheus.Labels{
				"address":       unbonding.DelegatorAddress,
				"denom":         *denom, // unbonding does not have denom in response for some reason
				"unbonded_from": unbonding.ValidatorAddress,
			}).Set(sum)
		}
	}()
	wg.Add(1)

	go func() {
		defer wg.Done()

		stakingClient := stakingtypes.NewQueryClient(grpcConn)
		stakingRes, err := stakingClient.Redelegations(
			context.Background(),
			&stakingtypes.QueryRedelegationsRequest{DelegatorAddr: myAddress.String()},
		)
		if err != nil {
			log.Error("Could not get redelegations for \"", address, "\", got error: ", err)
			return
		}

		for _, redelegation := range stakingRes.RedelegationResponses {
			var sum float64 = 0
			for _, entry := range redelegation.Entries {
				sum += float64(entry.Balance.Int64())
			}

			walletRedelegationGauge.With(prometheus.Labels{
				"address":          redelegation.Redelegation.DelegatorAddress,
				"denom":            *denom, // redelegation does not have denom in response for some reason
				"redelegated_from": redelegation.Redelegation.ValidatorSrcAddress,
				"redelegated_to":   redelegation.Redelegation.ValidatorDstAddress,
			}).Set(sum)
		}
	}()
	wg.Add(1)

	go func() {
		defer wg.Done()

		distributionClient := distributiontypes.NewQueryClient(grpcConn)
		distributionRes, err := distributionClient.DelegationTotalRewards(
			context.Background(),
			&distributiontypes.QueryDelegationTotalRewardsRequest{DelegatorAddress: myAddress.String()},
		)
		if err != nil {
			log.Error("Could not get rewards for \"", address, "\", got error: ", err)
			return
		}

		for _, reward := range distributionRes.Rewards {
			for _, entry := range reward.Reward {
				walletRewardsGauge.With(prometheus.Labels{
					"address":           address,
					"denom":             entry.Denom,
					"validator_address": reward.ValidatorAddress,
				}).Set(float64(entry.Amount.RoundInt64()))
			}
		}
	}()
	wg.Add(1)

	wg.Wait()

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
	log.Info("GET /metrics/wallet?address=", address)
}

func validatorHandler(w http.ResponseWriter, r *http.Request, grpcConn *grpc.ClientConn) {
	address := r.URL.Query().Get("address")
	myAddress, err := sdk.ValAddressFromBech32(address)
	if err != nil {
		log.Error("Could not get address for \"", address, "\", got error: ", err)
		return
	}

	validatorDelegationsGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_validator_delegations",
			Help: "Delegations of the Cosmos-based blockchain validator",
		},
		[]string{"address", "denom", "delegated_by"},
	)

	validatorTokensGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_validator_tokens",
			Help: "Tokens of the Cosmos-based blockchain validator",
		},
		[]string{"address"},
	)

	validatorDelegatorSharesGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_validator_delegators_shares",
			Help: "Delegators shares of the Cosmos-based blockchain validator",
		},
		[]string{"address"},
	)

	validatorCommissionRateGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cosmos_validator_commission_rate",
			Help: "Commission rate of the Cosmos-based blockchain validator",
		},
		[]string{"address"},
	)

	registry := prometheus.NewRegistry()
	registry.MustRegister(validatorDelegationsGauge)
	registry.MustRegister(validatorTokensGauge)
	registry.MustRegister(validatorDelegatorSharesGauge)
	registry.MustRegister(validatorCommissionRateGauge)

	var wg sync.WaitGroup

	go func() {
		defer wg.Done()

		stakingClient := stakingtypes.NewQueryClient(grpcConn)
		stakingRes, err := stakingClient.ValidatorDelegations(
			context.Background(),
			&stakingtypes.QueryValidatorDelegationsRequest{ValidatorAddr: myAddress.String()},
		)
		if err != nil {
			log.Error("Could not get delegations for \"", address, "\", got error: ", err)
			return
		}

		for _, delegation := range stakingRes.DelegationResponses {
			validatorDelegationsGauge.With(prometheus.Labels{
				"address":      delegation.Delegation.ValidatorAddress,
				"denom":        delegation.Balance.Denom,
				"delegated_by": delegation.Delegation.DelegatorAddress,
			}).Set(float64(delegation.Balance.Amount.Int64()))
		}
	}()
	wg.Add(1)

	go func() {
		defer wg.Done()

		stakingClient := stakingtypes.NewQueryClient(grpcConn)
		validator, err := stakingClient.Validator(
			context.Background(),
			&stakingtypes.QueryValidatorRequest{ValidatorAddr: myAddress.String()},
		)
		if err != nil {
			log.Error("Could not get delegations for \"", address, "\", got error: ", err)
			return
		}

		validatorTokensGauge.With(prometheus.Labels{
			"address": validator.Validator.OperatorAddress,
		}).Set(float64(validator.Validator.Tokens.Int64()))

		validatorDelegatorSharesGauge.With(prometheus.Labels{
			"address": validator.Validator.OperatorAddress,
		}).Set(float64(validator.Validator.DelegatorShares.RoundInt64()))

		// because cosmos's dec doesn't have .toFloat64() method or whatever and returns everything as int
		rate, err := strconv.ParseFloat(validator.Validator.Commission.CommissionRates.Rate.String(), 64)
		if err != nil {
			log.Error("Could not get commission rate for \"", address, "\", got error: ", err)
		} else {
			validatorCommissionRateGauge.With(prometheus.Labels{
				"address": validator.Validator.OperatorAddress,
			}).Set(rate)
		}
	}()
	wg.Add(1)

	wg.Wait()

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
	log.Info("GET /metrics/validator?address=", address)
}

func main() {
	flag.Parse()

	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(*prefix, *prefix+"pub")
	config.SetBech32PrefixForValidator(*prefix+"valoper", *prefix+"valoperpub")
	config.SetBech32PrefixForConsensusNode(*prefix+"valcons", *prefix+"valconspub")
	config.Seal()

	grpcConn, err := grpc.Dial(
		*nodeAddress,
		grpc.WithInsecure(),
	)
	if err != nil {
		panic(err)
	}

	defer grpcConn.Close()

	http.HandleFunc("/metrics/wallet", func(w http.ResponseWriter, r *http.Request) {
		walletHandler(w, r, grpcConn)
	})

	http.HandleFunc("/metrics/validator", func(w http.ResponseWriter, r *http.Request) {
		validatorHandler(w, r, grpcConn)
	})

	log.Info("Listening on ", *listenAddress)
	err = http.ListenAndServe(*listenAddress, nil)
	if err != nil {
		log.Fatal("Could not start application at ", *listenAddress, ", got error: ", err)
		panic(err)
	}
}

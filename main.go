package main

import (
    "context"
    "net/http"
    "google.golang.org/grpc"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "flag"
    "sync"

    log "github.com/sirupsen/logrus"
    sdk "github.com/cosmos/cosmos-sdk/types"
    banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
    stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

)

var prefix = flag.String("bech-prefix", "persistence", "Bech32 prefix for the network")
var denom = flag.String("denom", "uxprt", "Cosmos coin denom")
var listenAddress = flag.String("listen-address", ":9300", "The address this exporter would listen on")
var nodeAddress = flag.String("node", "localhost:9090", "RPC node address")

func walletHandler(w http.ResponseWriter, r *http.Request, grpcConn *grpc.ClientConn) {
    address :=  r.URL.Query().Get("address")
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

    walletUnbondingsGauge := prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "cosmos_wallet_unbondings",
            Help: "Unbondings of the Cosmos-based blockchain wallet",
        },
        []string{"address", "denom", "unbonded_from"},
    )

    registry := prometheus.NewRegistry()
    registry.MustRegister(walletBalanceGauge)
    registry.MustRegister(walletDelegationGauge)
    registry.MustRegister(walletUnbondingsGauge)

    
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
            "denom": bankRes.GetBalance().Denom,
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
                "address": address,
                "denom": delegation.Balance.Denom,
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
                "address": unbonding.DelegatorAddress,
                "denom": *denom, // unbonding does not have denom in response for some reason
                "unbonded_from": unbonding.ValidatorAddress,
            }).Set(sum)
        }
    }()
    wg.Add(1)

    wg.Wait()

    h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
    h.ServeHTTP(w, r)
    log.Info("GET /metrics/wallet?address=", address)
}

func main() {
    flag.Parse()

    config := sdk.GetConfig()
    config.SetBech32PrefixForAccount(*prefix, *prefix + "pub")
    config.SetBech32PrefixForValidator(*prefix + "valoper", *prefix + "valoperpub")
    config.SetBech32PrefixForConsensusNode(*prefix + "valcons", *prefix + "valconspub")
    config.Seal()

    grpcConn, err := grpc.Dial(
        *nodeAddress,
        grpc.WithInsecure(),
    )
    if err != nil {
        panic(err);
    }

    defer grpcConn.Close()

    http.HandleFunc("/metrics/wallet", func(w http.ResponseWriter, r *http.Request) {
        walletHandler(w, r, grpcConn)
    })

    log.Info("Listening on ", *listenAddress)
    err = http.ListenAndServe(*listenAddress, nil)
    if err != nil {
        log.Fatal("Could not start application at ", *listenAddress, ", got error: ", err)
        panic(err)
    }
}

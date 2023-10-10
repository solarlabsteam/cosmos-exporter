package main

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *service) SingleHandler(w http.ResponseWriter, r *http.Request) {
	requestStart := time.Now()

	sublogger := log.With().
		Str("request-id", uuid.New().String()).
		Logger()

	registry := prometheus.NewRegistry()
	generalMetrics := NewGeneralMetrics(registry)
	var validatorMetrics *ValidatorMetrics
	var paramsMetrics *ParamsMetrics
	var upgradeMetrics *UpgradeMetrics
	var walletMetrics *WalletMetrics
	var kujiOracleMetrics *KujiMetrics
	var proposalMetrics *ProposalsMetrics

	if len(s.Validators) > 0 {
		validatorMetrics = NewValidatorMetrics(registry)
	}
	if len(s.Wallets) > 0 {
		walletMetrics = NewWalletMetrics(registry)
	}
	if s.Params {
		paramsMetrics = NewParamsMetrics(registry)
	}
	if s.Upgrades {
		upgradeMetrics = NewUpgradeMetrics(registry)
	}
	if s.Oracle {
		kujiOracleMetrics = NewKujiMetrics(registry)
	}
	if s.Proposals {
		proposalMetrics = NewProposalsMetrics(registry)
	}

	var wg sync.WaitGroup

	getGeneralMetrics(&wg, &sublogger, generalMetrics, s)
	if paramsMetrics != nil {
		getParamsMetrics(&wg, &sublogger, paramsMetrics, s)
	}
	if upgradeMetrics != nil {
		getUpgradeMetrics(&wg, &sublogger, upgradeMetrics, s)
	}
	if len(s.Validators) > 0 {
		// use 2 groups.
		// the first group "val_wg" allows us to batch the initial validator call to get the moniker
		// the 'BasicMetrics' will then add a request to the outer wait 'wg'.
		// we ensure that all the requests are added by waiting for the 'val_wg' to finish before waiting on the 'wg'
		var val_wg sync.WaitGroup
		for _, validator := range s.Validators {
			valAddress, err := sdk.ValAddressFromBech32(validator)

			if err != nil {
				sublogger.Error().
					Str("address", validator).
					Err(err).
					Msg("Could not get validator address")

			} else {
				val_wg.Add(1)
				go func() {
					defer val_wg.Done()
					sublogger.Debug().Str("address", validator).Msg("Fetching validator details")

					getValidatorBasicMetrics(&wg, &sublogger, validatorMetrics, s, valAddress)
				}()

				if s.Oracle {
					sublogger.Debug().Str("address", validator).Msg("Fetching Kujira details")

					getKujiMetrics(&wg, &sublogger, kujiOracleMetrics, s, valAddress)
				}
			}
		}
		val_wg.Wait()
	}
	if len(s.Wallets) > 0 {
		for _, wallet := range s.Wallets {
			accAddress, err := sdk.AccAddressFromBech32(wallet)
			if err != nil {
				sublogger.Error().
					Str("address", wallet).
					Err(err).
					Msg("Could not get wallet address")
			} else {
				getWalletMetrics(&wg, &sublogger, walletMetrics, s, accAddress, false)
			}
		}
	}
	if s.Proposals {

		getProposalsMetrics(&wg, &sublogger, proposalMetrics, s, true)

	}
	wg.Wait()

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
	sublogger.Info().
		Str("method", "GET").
		Str("endpoint", "/metrics").
		Float64("request-time", time.Since(requestStart).Seconds()).
		Msg("Request processed")
}

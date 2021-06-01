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

	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func ParamsHandler(w http.ResponseWriter, r *http.Request, grpcConn *grpc.ClientConn) {
	requestStart := time.Now()

	sublogger := log.With().
		Str("request-id", uuid.New().String()).
		Logger()

	paramsMaxValidatorsGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "cosmos_params_max_validators",
			Help:        "Active set length",
			ConstLabels: ConstLabels,
		},
	)

	paramsUnbondingTimeGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "cosmos_params_unbonding_time",
			Help:        "Unbonding time, in seconds",
			ConstLabels: ConstLabels,
		},
	)

	paramsBlocksPerYearGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "cosmos_params_blocks_per_year",
			Help:        "Block per year",
			ConstLabels: ConstLabels,
		},
	)

	paramsGoalBondedGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "cosmos_params_goal_bonded",
			Help:        "Goal bonded",
			ConstLabels: ConstLabels,
		},
	)

	paramsInflationMinGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "cosmos_params_inflation_min",
			Help:        "Min inflation",
			ConstLabels: ConstLabels,
		},
	)

	paramsInflationMaxGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "cosmos_params_inflation_max",
			Help:        "Max inflation",
			ConstLabels: ConstLabels,
		},
	)

	paramsInflationRateChangeGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "cosmos_params_inflation_rate_change",
			Help:        "Inflation rate change",
			ConstLabels: ConstLabels,
		},
	)

	paramsDowntailJailDurationGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "cosmos_params_downtail_jail_duration",
			Help:        "Downtime jail duration, in seconds",
			ConstLabels: ConstLabels,
		},
	)

	paramsMinSignedPerWindowGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "cosmos_params_min_signed_per_window",
			Help:        "Minimal amount of blocks to sign per window to avoid slashing",
			ConstLabels: ConstLabels,
		},
	)

	paramsSignedBlocksWindowGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "cosmos_params_signed_blocks_window",
			Help:        "Signed blocks window",
			ConstLabels: ConstLabels,
		},
	)

	paramsSlashFractionDoubleSign := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "cosmos_params_slash_fraction_double_sign",
			Help:        "% of tokens to be slashed if double signing",
			ConstLabels: ConstLabels,
		},
	)

	paramsSlashFractionDowntime := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "cosmos_params_slash_fraction_downtime",
			Help:        "% of tokens to be slashed if downtime",
			ConstLabels: ConstLabels,
		},
	)

	paramsBaseProposerRewardGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "cosmos_params_base_proposer_reward",
			Help:        "Base proposer reward",
			ConstLabels: ConstLabels,
		},
	)

	paramsBonusProposerRewardGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "cosmos_params_bonus_proposer_reward",
			Help:        "Bonus proposer reward",
			ConstLabels: ConstLabels,
		},
	)
	paramsCommunityTaxGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "cosmos_params_community_tax",
			Help:        "Community tax",
			ConstLabels: ConstLabels,
		},
	)

	registry := prometheus.NewRegistry()
	registry.MustRegister(paramsMaxValidatorsGauge)
	registry.MustRegister(paramsUnbondingTimeGauge)
	registry.MustRegister(paramsBlocksPerYearGauge)
	registry.MustRegister(paramsInflationMinGauge)
	registry.MustRegister(paramsInflationMaxGauge)
	registry.MustRegister(paramsInflationRateChangeGauge)
	registry.MustRegister(paramsDowntailJailDurationGauge)
	registry.MustRegister(paramsMinSignedPerWindowGauge)
	registry.MustRegister(paramsSignedBlocksWindowGauge)
	registry.MustRegister(paramsSlashFractionDoubleSign)
	registry.MustRegister(paramsSlashFractionDowntime)
	registry.MustRegister(paramsBaseProposerRewardGauge)
	registry.MustRegister(paramsBonusProposerRewardGauge)
	registry.MustRegister(paramsCommunityTaxGauge)

	var wg sync.WaitGroup

	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started querying global staking params")
		queryStart := time.Now()

		stakingClient := stakingtypes.NewQueryClient(grpcConn)
		paramsResponse, err := stakingClient.Params(
			context.Background(),
			&stakingtypes.QueryParamsRequest{},
		)
		if err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not get global staking params")
			return
		}

		sublogger.Debug().
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying global staking params")

		paramsMaxValidatorsGauge.Set(float64(paramsResponse.Params.MaxValidators))
		paramsUnbondingTimeGauge.Set(paramsResponse.Params.UnbondingTime.Seconds())

	}()
	wg.Add(1)

	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started querying global mint params")
		queryStart := time.Now()

		mintClient := minttypes.NewQueryClient(grpcConn)
		paramsResponse, err := mintClient.Params(
			context.Background(),
			&minttypes.QueryParamsRequest{},
		)
		if err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not get global mint params")
			return
		}

		sublogger.Debug().
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying global mint params")

		paramsBlocksPerYearGauge.Set(float64(paramsResponse.Params.BlocksPerYear))

		// because cosmos's dec doesn't have .toFloat64() method or whatever and returns everything as int
		if value, err := strconv.ParseFloat(paramsResponse.Params.GoalBonded.String(), 64); err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not parse goal bonded")
		} else {
			paramsGoalBondedGauge.Set(value)
		}

		if value, err := strconv.ParseFloat(paramsResponse.Params.InflationMin.String(), 64); err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not parse inflation min")
		} else {
			paramsInflationMinGauge.Set(value)
		}

		if value, err := strconv.ParseFloat(paramsResponse.Params.InflationMax.String(), 64); err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not parse inflation min")
		} else {
			paramsInflationMaxGauge.Set(value)
		}

		if value, err := strconv.ParseFloat(paramsResponse.Params.InflationRateChange.String(), 64); err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not parse inflation rate change")
		} else {
			paramsInflationRateChangeGauge.Set(value)
		}
	}()
	wg.Add(1)

	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started querying global slashing params")
		queryStart := time.Now()

		slashingClient := slashingtypes.NewQueryClient(grpcConn)
		paramsResponse, err := slashingClient.Params(
			context.Background(),
			&slashingtypes.QueryParamsRequest{},
		)
		if err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not get global slashing params")
			return
		}

		sublogger.Debug().
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying global slashing params")

		paramsDowntailJailDurationGauge.Set(paramsResponse.Params.DowntimeJailDuration.Seconds())
		paramsSignedBlocksWindowGauge.Set(float64(paramsResponse.Params.SignedBlocksWindow))

		if value, err := strconv.ParseFloat(paramsResponse.Params.MinSignedPerWindow.String(), 64); err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not parse min signed per window")
		} else {
			paramsMinSignedPerWindowGauge.Set(value)
		}

		if value, err := strconv.ParseFloat(paramsResponse.Params.SlashFractionDoubleSign.String(), 64); err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not parse slash fraction double sign")
		} else {
			paramsSlashFractionDoubleSign.Set(value)
		}

		if value, err := strconv.ParseFloat(paramsResponse.Params.SlashFractionDowntime.String(), 64); err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not parse slash fraction downtime")
		} else {
			paramsSlashFractionDowntime.Set(value)
		}
	}()
	wg.Add(1)

	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started querying global distribution params")
		queryStart := time.Now()

		distributionClient := distributiontypes.NewQueryClient(grpcConn)
		paramsResponse, err := distributionClient.Params(
			context.Background(),
			&distributiontypes.QueryParamsRequest{},
		)
		if err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not get global distribution params")
			return
		}

		sublogger.Debug().
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying global distribution params")

		// because cosmos's dec doesn't have .toFloat64() method or whatever and returns everything as int
		if value, err := strconv.ParseFloat(paramsResponse.Params.BaseProposerReward.String(), 64); err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not parse base proposer reward")
		} else {
			paramsBaseProposerRewardGauge.Set(value)
		}

		if value, err := strconv.ParseFloat(paramsResponse.Params.BonusProposerReward.String(), 64); err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not parse bonus proposer reward")
		} else {
			paramsBonusProposerRewardGauge.Set(value)
		}

		if value, err := strconv.ParseFloat(paramsResponse.Params.CommunityTax.String(), 64); err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not parse community rate")
		} else {
			paramsCommunityTaxGauge.Set(value)
		}
	}()
	wg.Add(1)

	wg.Wait()

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
	sublogger.Info().
		Str("method", "GET").
		Str("endpoint", "/metrics/params").
		Float64("request-time", time.Since(requestStart).Seconds()).
		Msg("Request processed")
}

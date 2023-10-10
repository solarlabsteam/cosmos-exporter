package main

import (
	"context"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/rs/zerolog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type ParamsMetrics struct {
	maxValidatorsGauge        prometheus.Gauge
	unbondingTimeGauge        prometheus.Gauge
	blocksPerYearGauge        prometheus.Gauge
	goalBondedGauge           prometheus.Gauge
	inflationMinGauge         prometheus.Gauge
	inflationMaxGauge         prometheus.Gauge
	inflationRateChangeGauge  prometheus.Gauge
	downtimeJailDurationGauge prometheus.Gauge
	minSignedPerWindowGauge   prometheus.Gauge
	signedBlocksWindowGauge   prometheus.Gauge
	slashFractionDoubleSign   prometheus.Gauge
	slashFractionDowntime     prometheus.Gauge
	baseProposerRewardGauge   prometheus.Gauge
	bonusProposerRewardGauge  prometheus.Gauge
	communityTaxGauge         prometheus.Gauge
}

func NewParamsMetrics(reg prometheus.Registerer) *ParamsMetrics {
	m := &ParamsMetrics{
		maxValidatorsGauge: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name:        "cosmos_params_max_validators",
				Help:        "Active set length",
				ConstLabels: ConstLabels,
			},
		),
		unbondingTimeGauge: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name:        "cosmos_params_unbonding_time",
				Help:        "Unbonding time, in seconds",
				ConstLabels: ConstLabels,
			},
		),
		blocksPerYearGauge: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name:        "cosmos_params_blocks_per_year",
				Help:        "Block per year",
				ConstLabels: ConstLabels,
			},
		),
		goalBondedGauge: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name:        "cosmos_params_goal_bonded",
				Help:        "Goal bonded",
				ConstLabels: ConstLabels,
			},
		),
		inflationMinGauge: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name:        "cosmos_params_inflation_min",
				Help:        "Min inflation",
				ConstLabels: ConstLabels,
			},
		),
		inflationMaxGauge: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name:        "cosmos_params_inflation_max",
				Help:        "Max inflation",
				ConstLabels: ConstLabels,
			},
		),
		inflationRateChangeGauge: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name:        "cosmos_params_inflation_rate_change",
				Help:        "Inflation rate change",
				ConstLabels: ConstLabels,
			},
		),
		downtimeJailDurationGauge: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name:        "cosmos_params_downtime_jail_duration",
				Help:        "Downtime jail duration, in seconds",
				ConstLabels: ConstLabels,
			},
		),
		minSignedPerWindowGauge: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name:        "cosmos_params_min_signed_per_window",
				Help:        "Minimal amount of blocks to sign per window to avoid slashing",
				ConstLabels: ConstLabels,
			},
		),
		signedBlocksWindowGauge: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name:        "cosmos_params_signed_blocks_window",
				Help:        "Signed blocks window",
				ConstLabels: ConstLabels,
			},
		),
		slashFractionDoubleSign: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name:        "cosmos_params_slash_fraction_double_sign",
				Help:        "% of tokens to be slashed if double signing",
				ConstLabels: ConstLabels,
			},
		),
		slashFractionDowntime: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name:        "cosmos_params_slash_fraction_downtime",
				Help:        "% of tokens to be slashed if downtime",
				ConstLabels: ConstLabels,
			},
		),
		baseProposerRewardGauge: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name:        "cosmos_params_base_proposer_reward",
				Help:        "Base proposer reward",
				ConstLabels: ConstLabels,
			},
		),
		bonusProposerRewardGauge: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name:        "cosmos_params_bonus_proposer_reward",
				Help:        "Bonus proposer reward",
				ConstLabels: ConstLabels,
			},
		),
		communityTaxGauge: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name:        "cosmos_params_community_tax",
				Help:        "Community tax",
				ConstLabels: ConstLabels,
			},
		),
	}

	reg.MustRegister(m.maxValidatorsGauge)
	reg.MustRegister(m.unbondingTimeGauge)
	reg.MustRegister(m.blocksPerYearGauge)
	reg.MustRegister(m.inflationMinGauge)
	reg.MustRegister(m.inflationMaxGauge)
	reg.MustRegister(m.inflationRateChangeGauge)
	reg.MustRegister(m.downtimeJailDurationGauge)
	reg.MustRegister(m.minSignedPerWindowGauge)
	reg.MustRegister(m.signedBlocksWindowGauge)
	reg.MustRegister(m.slashFractionDoubleSign)
	reg.MustRegister(m.slashFractionDowntime)
	reg.MustRegister(m.baseProposerRewardGauge)
	reg.MustRegister(m.bonusProposerRewardGauge)
	reg.MustRegister(m.communityTaxGauge)

	return m
}
func getParamsMetrics(wg *sync.WaitGroup, sublogger *zerolog.Logger, metrics *ParamsMetrics, s *service) {

	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started querying global staking params")
		queryStart := time.Now()

		stakingClient := stakingtypes.NewQueryClient(s.grpcConn)
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

		metrics.maxValidatorsGauge.Set(float64(paramsResponse.Params.MaxValidators))
		metrics.unbondingTimeGauge.Set(paramsResponse.Params.UnbondingTime.Seconds())
	}()
	wg.Add(1)

	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started querying global mint params")
		queryStart := time.Now()

		mintClient := minttypes.NewQueryClient(s.grpcConn)
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

		metrics.blocksPerYearGauge.Set(float64(paramsResponse.Params.BlocksPerYear))

		// because cosmos's dec doesn't have .toFloat64() method or whatever and returns everything as int
		if value, err := strconv.ParseFloat(paramsResponse.Params.GoalBonded.String(), 64); err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not parse goal bonded")
		} else {
			metrics.goalBondedGauge.Set(value)
		}

		if value, err := strconv.ParseFloat(paramsResponse.Params.InflationMin.String(), 64); err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not parse inflation min")
		} else {
			metrics.inflationMinGauge.Set(value)
		}

		if value, err := strconv.ParseFloat(paramsResponse.Params.InflationMax.String(), 64); err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not parse inflation min")
		} else {
			metrics.inflationMaxGauge.Set(value)
		}

		if value, err := strconv.ParseFloat(paramsResponse.Params.InflationRateChange.String(), 64); err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not parse inflation rate change")
		} else {
			metrics.inflationRateChangeGauge.Set(value)
		}
	}()
	wg.Add(1)

	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started querying global slashing params")
		queryStart := time.Now()

		slashingClient := slashingtypes.NewQueryClient(s.grpcConn)
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

		metrics.downtimeJailDurationGauge.Set(paramsResponse.Params.DowntimeJailDuration.Seconds())
		metrics.signedBlocksWindowGauge.Set(float64(paramsResponse.Params.SignedBlocksWindow))

		if value, err := strconv.ParseFloat(paramsResponse.Params.MinSignedPerWindow.String(), 64); err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not parse min signed per window")
		} else {
			metrics.minSignedPerWindowGauge.Set(value)
		}

		if value, err := strconv.ParseFloat(paramsResponse.Params.SlashFractionDoubleSign.String(), 64); err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not parse slash fraction double sign")
		} else {
			metrics.slashFractionDoubleSign.Set(value)
		}

		if value, err := strconv.ParseFloat(paramsResponse.Params.SlashFractionDowntime.String(), 64); err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not parse slash fraction downtime")
		} else {
			metrics.slashFractionDowntime.Set(value)
		}
	}()
	wg.Add(1)

	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started querying global distribution params")
		queryStart := time.Now()

		distributionClient := distributiontypes.NewQueryClient(s.grpcConn)
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
			metrics.baseProposerRewardGauge.Set(value)
		}

		if value, err := strconv.ParseFloat(paramsResponse.Params.BonusProposerReward.String(), 64); err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not parse bonus proposer reward")
		} else {
			metrics.bonusProposerRewardGauge.Set(value)
		}

		if value, err := strconv.ParseFloat(paramsResponse.Params.CommunityTax.String(), 64); err != nil {
			sublogger.Error().
				Err(err).
				Msg("Could not parse community rate")
		} else {
			metrics.communityTaxGauge.Set(value)
		}
	}()
	wg.Add(1)

}
func (s *service) ParamsHandler(w http.ResponseWriter, r *http.Request) {
	requestStart := time.Now()

	sublogger := log.With().
		Str("request-id", uuid.New().String()).
		Logger()

	registry := prometheus.NewRegistry()
	paramsMetrics := NewParamsMetrics(registry)

	var wg sync.WaitGroup
	getParamsMetrics(&wg, &sublogger, paramsMetrics, s)

	wg.Wait()

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
	sublogger.Info().
		Str("method", "GET").
		Str("endpoint", "/metrics/params").
		Float64("request-time", time.Since(requestStart).Seconds()).
		Msg("Request processed")
}

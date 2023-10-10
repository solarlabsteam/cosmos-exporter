package main

import (
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/rs/zerolog"
	"net/http"
	"sync"
	"time"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type ProposalsMetrics struct {
	proposalsGauge *prometheus.GaugeVec
}

func NewProposalsMetrics(reg prometheus.Registerer) *ProposalsMetrics {
	m := &ProposalsMetrics{
		proposalsGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "cosmos_proposals",
				Help:        "Proposals of Cosmos-based blockchain",
				ConstLabels: ConstLabels,
			},
			[]string{"title", "status", "voting_start_time", "voting_end_time"},
		),
	}
	reg.MustRegister(m.proposalsGauge)
	return m
}
func getProposalsMetrics(wg *sync.WaitGroup, sublogger *zerolog.Logger, metrics *ProposalsMetrics, s *service, activeOnly bool) {

	wg.Add(1)
	go func() {
		defer wg.Done()

		var proposals []govtypes.Proposal

		sublogger.Debug().Msg("Started querying proposals")
		queryStart := time.Now()

		govClient := govtypes.NewQueryClient(s.grpcConn)

		var propReq govtypes.QueryProposalsRequest
		if activeOnly {
			propReq = govtypes.QueryProposalsRequest{ProposalStatus: govtypes.StatusVotingPeriod, Pagination: &query.PageRequest{Reverse: true}}
		} else {
			propReq = govtypes.QueryProposalsRequest{Pagination: &query.PageRequest{Reverse: true}}
		}
		proposalsResponse, err := govClient.Proposals(
			context.Background(),
			&propReq,
		)
		if err != nil {
			sublogger.Error().Err(err).Msg("Could not get proposals")
			return
		}

		sublogger.Debug().
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying proposals")
		proposals = proposalsResponse.Proposals

		sublogger.Debug().
			Int("proposalsLength", len(proposals)).
			Msg("Proposals info")

		cdcRegistry := codectypes.NewInterfaceRegistry()
		cdc := codec.NewProtoCodec(cdcRegistry)
		for _, proposal := range proposals {

			var content govtypes.TextProposal
			err := cdc.Unmarshal(proposal.Content.Value, &content)

			if err != nil {
				sublogger.Error().
					Str("proposal_id", fmt.Sprint(proposal.ProposalId)).
					Err(err).
					Msg("Could not parse proposal content")
			}

			metrics.proposalsGauge.With(prometheus.Labels{
				"title":             content.Title,
				"status":            proposal.Status.String(),
				"voting_start_time": proposal.VotingStartTime.String(),
				"voting_end_time":   proposal.VotingEndTime.String(),
			}).Set(float64(proposal.ProposalId))

		}
	}()

}
func (s *service) ProposalsHandler(w http.ResponseWriter, r *http.Request) {
	requestStart := time.Now()

	sublogger := log.With().
		Str("request-id", uuid.New().String()).
		Logger()

	registry := prometheus.NewRegistry()
	proposalsMetrics := NewProposalsMetrics(registry)

	var wg sync.WaitGroup

	getProposalsMetrics(&wg, &sublogger, proposalsMetrics, s, false)

	wg.Wait()
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
	sublogger.Info().
		Str("method", "GET").
		Str("endpoint", "/metrics/proposals").
		Float64("request-time", time.Since(requestStart).Seconds()).
		Msg("Request processed")
}

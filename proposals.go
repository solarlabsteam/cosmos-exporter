package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

func ProposalsHandler(w http.ResponseWriter, r *http.Request, grpcConn *grpc.ClientConn) {
	requestStart := time.Now()

	sublogger := log.With().
		Str("request-id", uuid.New().String()).
		Logger()

	proposalsGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:        "cosmos_proposals",
			Help:        "Proposals of Cosmos-based blockchain",
			ConstLabels: ConstLabels,
		},
		[]string{"title", "status", "voting_start_time", "voting_end_time"},
	)

	registry := prometheus.NewRegistry()
	registry.MustRegister(proposalsGauge)

	var proposals []govtypes.Proposal

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		sublogger.Debug().Msg("Started querying proposals")
		queryStart := time.Now()

		govClient := govtypes.NewQueryClient(grpcConn)
		proposalsResponse, err := govClient.Proposals(
			context.Background(),
			&govtypes.QueryProposalsRequest{},
		)
		if err != nil {
			sublogger.Error().Err(err).Msg("Could not get proposals")
			return
		}

		sublogger.Debug().
			Float64("request-time", time.Since(queryStart).Seconds()).
			Msg("Finished querying proposals")
		proposals = proposalsResponse.Proposals
	}()

	wg.Wait()

	sublogger.Debug().
		Int("proposalsLength", len(proposals)).
		Msg("Proposals info")

	cdcRegistry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(cdcRegistry)
	for _, proposal := range proposals {
		var content govtypes.TextProposal
		err := cdc.UnmarshalBinaryBare(proposal.Content.Value, &content)

		if err != nil {
			sublogger.Error().
				Str("proposal_id", fmt.Sprint(proposal.ProposalId)).
				Err(err).
				Msg("Could not parse proposal content")
		}

		proposalsGauge.With(prometheus.Labels{
			"title":             content.Title,
			"status":            proposal.Status.String(),
			"voting_start_time": proposal.VotingStartTime.String(),
			"voting_end_time":   proposal.VotingEndTime.String(),
		}).Set(float64(proposal.ProposalId))
	}

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
	sublogger.Info().
		Str("method", "GET").
		Str("endpoint", "/metrics/proposals").
		Float64("request-time", time.Since(requestStart).Seconds()).
		Msg("Request processed")
}

package main

import (
	"flag"
	"net/http"
	"os"

	"google.golang.org/grpc"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/rs/zerolog"
)

var PrefixFlag = flag.String("bech-prefix", "persistence", "Bech32 global prefix")

// some networks, like Iris, have the different prefixes for address, validator and consensus node
var AccountPrefixFlag = flag.String("bech-account-prefix", "", "Bech32 account prefix")
var AccountPubkeyPrefixFlag = flag.String("bech-account-pubkey-prefix", "", "Bech32 pubkey account prefix")
var ValidatorPrefixFlag = flag.String("bech-validator-prefix", "", "Bech32 validator prefix")
var ValidatorPubkeyPrefixFlag = flag.String("bech-validator-pubkey-prefix", "", "Bech32 pubkey validator prefix")
var ConsensusNodePrefixFlag = flag.String("bech-consensus-node-prefix", "", "Bech32 consensus node prefix")
var ConsensusNodePubkeyPrefixFlag = flag.String("bech-consensus-node-pubkey-prefix", "", "Bech32 pubkey consensus node prefix")

var Denom = flag.String("denom", "uxprt", "Cosmos coin denom")
var ListenAddress = flag.String("listen-address", ":9300", "The address this exporter would listen on")
var NodeAddress = flag.String("node", "localhost:9090", "RPC node address")
var LogLevel = flag.String("log-level", "info", "Logging level")

var log = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()

var AccountPrefix string
var AccountPubkeyPrefix string
var ValidatorPrefix string
var ValidatorPubkeyPrefix string
var ConsensusNodePrefix string
var ConsensusNodePubkeyPrefix string

func main() {
	flag.Parse()

	logLevel, err := zerolog.ParseLevel(*LogLevel)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not parse log level")
	}

	zerolog.SetGlobalLevel(logLevel)

	if *AccountPrefixFlag == "" {
		AccountPrefix = *PrefixFlag
	} else {
		AccountPrefix = *AccountPrefixFlag
	}

	if *AccountPubkeyPrefixFlag == "" {
		AccountPubkeyPrefix = *PrefixFlag + "pub"
	} else {
		AccountPubkeyPrefix = *PrefixFlag
	}

	if *ValidatorPrefixFlag == "" {
		ValidatorPrefix = *PrefixFlag + "valoper"
	} else {
		ValidatorPrefix = *ValidatorPrefixFlag
	}

	if *ValidatorPubkeyPrefixFlag == "" {
		ValidatorPubkeyPrefix = *PrefixFlag + "valoperpub"
	} else {
		ValidatorPubkeyPrefix = *ValidatorPubkeyPrefixFlag
	}

	if *ConsensusNodePrefixFlag == "" {
		ConsensusNodePrefix = *PrefixFlag + "valcons"
	} else {
		ConsensusNodePrefix = *ConsensusNodePrefixFlag
	}

	if *ConsensusNodePubkeyPrefixFlag == "" {
		ConsensusNodePubkeyPrefix = *PrefixFlag + "valconspub"
	} else {
		ConsensusNodePubkeyPrefix = *ConsensusNodePrefixFlag
	}

	log.Info().
		Str("--bech-account-prefix", AccountPrefix).
		Str("--bech-account-pubkey-prefix", AccountPubkeyPrefix).
		Str("--bech-validator-prefix", ValidatorPrefix).
		Str("--bech-validator-pubkey-prefix", ValidatorPubkeyPrefix).
		Str("--bech-consensus-node-prefix", ConsensusNodePrefix).
		Str("--bech-consensus-node-pubkey-prefix", ConsensusNodePubkeyPrefix).
		Str("--denom", *Denom).
		Str("--listen-address", *ListenAddress).
		Str("--node", *NodeAddress).
		Str("--log-level", *LogLevel).
		Msg("Started with following parameters")

	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(AccountPrefix, AccountPubkeyPrefix)
	config.SetBech32PrefixForValidator(ValidatorPrefix, ValidatorPubkeyPrefix)
	config.SetBech32PrefixForConsensusNode(ConsensusNodePrefix, ConsensusNodePubkeyPrefix)
	config.Seal()

	grpcConn, err := grpc.Dial(
		*NodeAddress,
		grpc.WithInsecure(),
	)
	if err != nil {
		panic(err)
	}

	defer grpcConn.Close()

	http.HandleFunc("/metrics/wallet", func(w http.ResponseWriter, r *http.Request) {
		WalletHandler(w, r, grpcConn)
	})

	http.HandleFunc("/metrics/validator", func(w http.ResponseWriter, r *http.Request) {
		ValidatorHandler(w, r, grpcConn)
	})

	http.HandleFunc("/metrics/validators", func(w http.ResponseWriter, r *http.Request) {
		ValidatorsHandler(w, r, grpcConn)
	})

	log.Info().Str("address", *ListenAddress).Msg("Listening")
	err = http.ListenAndServe(*ListenAddress, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not start application")
	}
}

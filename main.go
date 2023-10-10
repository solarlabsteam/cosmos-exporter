package main

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	tmrpc "github.com/tendermint/tendermint/rpc/client/http"
	"google.golang.org/grpc"
)

var (
	ConfigPath string

	Denom         string
	ListenAddress string
	NodeAddress   string
	TendermintRPC string
	LogLevel      string
	JsonOutput    bool
	Limit         uint64

	Prefix                    string
	AccountPrefix             string
	AccountPubkeyPrefix       string
	ValidatorPrefix           string
	ValidatorPubkeyPrefix     string
	ConsensusNodePrefix       string
	ConsensusNodePubkeyPrefix string

	ChainID          string
	ConstLabels      map[string]string
	DenomCoefficient float64
	DenomExponent    uint64

	// SingleReq bundle up multiple requests into a single /metrics
	SingleReq  bool
	Wallets    []string
	Validators []string
	Oracle     bool
	Upgrades   bool
	Proposals  bool
	Params     bool
)

type service struct {
	grpcConn   *grpc.ClientConn
	tmRPC      *tmrpc.HTTP
	Wallets    []string
	Validators []string
	Oracle     bool
	Upgrades   bool
	Proposals  bool
	Params     bool
}

var log = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()

var rootCmd = &cobra.Command{
	Use:  "cosmos-exporter",
	Long: "Scrape the data about the validators set, specific validators or wallets in the Cosmos network.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if ConfigPath == "" {
			setBechPrefixes(cmd)
			return nil
		}

		viper.SetConfigFile(ConfigPath)
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				log.Info().Err(err).Msg("Error reading config file")
				return err
			}
		}

		// Credits to https://carolynvanslyck.com/blog/2020/08/sting-of-the-viper/
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if !f.Changed && viper.IsSet(f.Name) {
				val := viper.Get(f.Name)
				if err := cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val)); err != nil {
					log.Fatal().Err(err).Msg("Could not set flag")
				}
			}
		})

		setBechPrefixes(cmd)
		/*
			SingleReq, err := cmd.Flags().GetBool("single")
			if err != nil {
				log.Fatal().Err(err).Msg("Could not set flag")
				return err
			}

			Oracle, err := cmd.Flags().GetBool("oracle")
			if err != nil {
				log.Fatal().Err(err).Msg("Could not set flag")
				return err
			}
			Upgrades, err := cmd.Flags().GetBool("upgrades")
			if err != nil {
				log.Fatal().Err(err).Msg("Could not set flag")
				return err
			}
			Proposals, err := cmd.Flags().GetBool("proposals")
			if err != nil {
				log.Fatal().Err(err).Msg("Could not set flag")
				return err
			}

			Params, err := cmd.Flags().GetBool("params")
			if err != nil {
				log.Fatal().Err(err).Msg("Could not set flag")
				return err
			}
			Wallets, err := cmd.Flags().GetStringArray("wallets")
			if err != nil {
				log.Fatal().Err(err).Msg("Could not set flag")
				return err
			}
			Validators, err := cmd.Flags().GetStringArray("validators")
			if err != nil {
				log.Fatal().Err(err).Msg("Could not set flag")
				return err
			}
		*/
		return nil
	},
	Run: Execute,
}

func setBechPrefixes(cmd *cobra.Command) {
	if flag, err := cmd.Flags().GetString("bech-account-prefix"); flag != "" && err == nil {
		AccountPrefix = flag
	} else {
		AccountPrefix = Prefix
	}

	if flag, err := cmd.Flags().GetString("bech-account-pubkey-prefix"); flag != "" && err == nil {
		AccountPubkeyPrefix = flag
	} else {
		AccountPubkeyPrefix = Prefix + "pub"
	}

	if flag, err := cmd.Flags().GetString("bech-validator-prefix"); flag != "" && err == nil {
		ValidatorPrefix = flag
	} else {
		ValidatorPrefix = Prefix + "valoper"
	}

	if flag, err := cmd.Flags().GetString("bech-validator-pubkey-prefix"); flag != "" && err == nil {
		ValidatorPubkeyPrefix = flag
	} else {
		ValidatorPubkeyPrefix = Prefix + "valoperpub"
	}

	if flag, err := cmd.Flags().GetString("bech-consensus-node-prefix"); flag != "" && err == nil {
		ConsensusNodePrefix = flag
	} else {
		ConsensusNodePrefix = Prefix + "valcons"
	}

	if flag, err := cmd.Flags().GetString("bech-consensus-node-pubkey-prefix"); flag != "" && err == nil {
		ConsensusNodePubkeyPrefix = flag
	} else {
		ConsensusNodePubkeyPrefix = Prefix + "valconspub"
	}
}

func Execute(_ *cobra.Command, _ []string) {
	logLevel, err := zerolog.ParseLevel(LogLevel)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not parse log level")
	}

	if JsonOutput {
		log = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}

	zerolog.SetGlobalLevel(logLevel)

	log.Info().
		Str("--bech-account-prefix", AccountPrefix).
		Str("--bech-account-pubkey-prefix", AccountPubkeyPrefix).
		Str("--bech-validator-prefix", ValidatorPrefix).
		Str("--bech-validator-pubkey-prefix", ValidatorPubkeyPrefix).
		Str("--bech-consensus-node-prefix", ConsensusNodePrefix).
		Str("--bech-consensus-node-pubkey-prefix", ConsensusNodePubkeyPrefix).
		Str("--denom", Denom).
		Str("--denom-cofficient", fmt.Sprintf("%f", DenomCoefficient)).
		Str("--denom-exponent", fmt.Sprintf("%d", DenomExponent)).
		Str("--listen-address", ListenAddress).
		Str("--node", NodeAddress).
		Str("--log-level", LogLevel).
		Str("--single", fmt.Sprintf("%t", SingleReq)).
		Str("--wallets", strings.Join(Wallets[:], ",")).
		Str("--validators", strings.Join(Validators[:], ",")).
		Str("--oracle", fmt.Sprintf("%t", Oracle)).
		Str("--proposals", fmt.Sprintf("%t", Proposals)).
		Str("--params", fmt.Sprintf("%t", Params)).
		Str("--upgrades", fmt.Sprintf("%t", Upgrades)).
		Msg("Started with following parameters")

	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(AccountPrefix, AccountPubkeyPrefix)
	config.SetBech32PrefixForValidator(ValidatorPrefix, ValidatorPubkeyPrefix)
	config.SetBech32PrefixForConsensusNode(ConsensusNodePrefix, ConsensusNodePubkeyPrefix)
	config.Seal()

	s := &service{}

	// Setup gRPC connection
	s.grpcConn, err = grpc.Dial(
		NodeAddress,
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not connect to gRPC node")
	}
	defer func(grpcConn *grpc.ClientConn) {
		err := grpcConn.Close()
		if err != nil {
			log.Fatal().Err(err).Msg("Could not close gRPC client")
		}
	}(s.grpcConn)

	// Setup Tendermint RPC connection
	s.tmRPC, err = tmrpc.New(TendermintRPC, "/websocket")
	if err != nil {
		log.Fatal().Err(err).Msg("Could not create Tendermint client")
	}
	s.setChainID()
	s.setDenom()
	s.Params = Params
	s.Wallets = Wallets
	s.Validators = Validators
	s.Proposals = Proposals
	s.Oracle = Oracle
	s.Params = Params
	s.Upgrades = Upgrades

	if SingleReq {
		log.Info().Msg("Starting Single Mode")
		http.HandleFunc("/metrics", s.SingleHandler)
	}
	http.HandleFunc("/metrics/wallet", s.WalletHandler)
	http.HandleFunc("/metrics/validator", s.ValidatorHandler)
	http.HandleFunc("/metrics/validators", s.ValidatorsHandler)
	http.HandleFunc("/metrics/params", s.ParamsHandler)
	http.HandleFunc("/metrics/general", s.GeneralHandler)
	http.HandleFunc("/metrics/kujira", s.KujiraMetricHandler)
	http.HandleFunc("/metrics/delegator", s.DelegatorHandler)
	http.HandleFunc("/metrics/proposals", s.ProposalsHandler)
	http.HandleFunc("/metrics/upgrade", s.UpgradeHandler)

	log.Info().Str("address", ListenAddress).Msg("Listening")
	err = http.ListenAndServe(ListenAddress, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not start application")
	}
}

func (s *service) setChainID() {
	status, err := s.tmRPC.Status(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("Could not query Tendermint status")
	}

	log.Info().Str("network", status.NodeInfo.Network).Msg("Got network status from Tendermint")
	ChainID = status.NodeInfo.Network
	ConstLabels = map[string]string{
		"chain_id": ChainID,
	}
}

func (s *service) setDenom() {
	// if --denom and (--denom-coefficient or --denom-exponent) are provided, use them
	// instead of fetching them via gRPC. Can be useful for networks like osmosis.
	if isUserProvidedAndHandled := checkAndHandleDenomInfoProvidedByUser(); isUserProvidedAndHandled {
		return
	}

	bankClient := banktypes.NewQueryClient(s.grpcConn)
	denoms, err := bankClient.DenomsMetadata(
		context.Background(),
		&banktypes.QueryDenomsMetadataRequest{},
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Error querying denom")
	}

	if len(denoms.Metadatas) == 0 {
		log.Fatal().Msg("No denom infos. Try running the binary with --denom and --denom-coefficient to set them manually.")
	}

	metadata := denoms.Metadatas[0] // always using the first one
	if Denom == "" {                // using display currency
		Denom = metadata.Display
	}

	for _, unit := range metadata.DenomUnits {
		log.Debug().
			Str("denom", unit.Denom).
			Uint32("exponent", unit.Exponent).
			Msg("Denom info")
		if unit.Denom == Denom {
			DenomCoefficient = math.Pow10(int(unit.Exponent))
			log.Info().
				Str("denom", Denom).
				Float64("coefficient", DenomCoefficient).
				Msg("Got denom info")
			return
		}
	}

	log.Fatal().Msg("Could not find the denom info")
}

func checkAndHandleDenomInfoProvidedByUser() bool {

	if Denom != "" {
		if DenomCoefficient != 1 && DenomExponent != 0 {
			log.Fatal().Msg("denom-coefficient and denom-exponent are both provided. Must provide only one")
		}

		if DenomCoefficient != 1 {
			log.Info().
				Str("denom", Denom).
				Float64("coefficient", DenomCoefficient).
				Msg("Using provided denom and coefficient.")
			return true
		}

		if DenomExponent != 0 {
			DenomCoefficient = math.Pow10(int(DenomExponent))
			log.Info().
				Str("denom", Denom).
				Uint64("exponent", DenomExponent).
				Float64("calculated coefficient", DenomCoefficient).
				Msg("Using provided denom and denom exponent and calculating coefficient.")
			return true
		}

		return false
	}

	return false

}

func main() {
	rootCmd.PersistentFlags().StringVar(&ConfigPath, "config", "", "Config file path")
	rootCmd.PersistentFlags().StringVar(&Denom, "denom", "", "Cosmos coin denom")
	rootCmd.PersistentFlags().Float64Var(&DenomCoefficient, "denom-coefficient", 1, "Denom coefficient")
	rootCmd.PersistentFlags().Uint64Var(&DenomExponent, "denom-exponent", 0, "Denom exponent")
	rootCmd.PersistentFlags().StringVar(&ListenAddress, "listen-address", ":9300", "The address this exporter would listen on")
	rootCmd.PersistentFlags().StringVar(&NodeAddress, "node", "localhost:9090", "RPC node address")
	rootCmd.PersistentFlags().StringVar(&LogLevel, "log-level", "info", "Logging level")
	rootCmd.PersistentFlags().Uint64Var(&Limit, "limit", 1000, "Pagination limit for gRPC requests")
	rootCmd.PersistentFlags().StringVar(&TendermintRPC, "tendermint-rpc", "http://localhost:26657", "Tendermint RPC address")
	rootCmd.PersistentFlags().BoolVar(&JsonOutput, "json", false, "Output logs as JSON")

	// some networks, like Iris, have the different prefixes for address, validator and consensus node
	rootCmd.PersistentFlags().StringVar(&Prefix, "bech-prefix", "persistence", "Bech32 global prefix")
	rootCmd.PersistentFlags().StringVar(&AccountPrefix, "bech-account-prefix", "", "Bech32 account prefix")
	rootCmd.PersistentFlags().StringVar(&AccountPubkeyPrefix, "bech-account-pubkey-prefix", "", "Bech32 pubkey account prefix")
	rootCmd.PersistentFlags().StringVar(&ValidatorPrefix, "bech-validator-prefix", "", "Bech32 validator prefix")
	rootCmd.PersistentFlags().StringVar(&ValidatorPubkeyPrefix, "bech-validator-pubkey-prefix", "", "Bech32 pubkey validator prefix")
	rootCmd.PersistentFlags().StringVar(&ConsensusNodePrefix, "bech-consensus-node-prefix", "", "Bech32 consensus node prefix")
	rootCmd.PersistentFlags().StringVar(&ConsensusNodePubkeyPrefix, "bech-consensus-node-pubkey-prefix", "", "Bech32 pubkey consensus node prefix")
	rootCmd.PersistentFlags().BoolVar(&SingleReq, "single", false, "serve info in a single call to /metrics")
	rootCmd.PersistentFlags().BoolVar(&Oracle, "oracle", false, "serve oracle info in the single call to /metrics")
	rootCmd.PersistentFlags().BoolVar(&Upgrades, "upgrades", false, "serve upgrade info in the single call to /metrics")
	rootCmd.PersistentFlags().BoolVar(&Proposals, "proposals", false, "serve active proposal info in the single call to /metrics")
	rootCmd.PersistentFlags().BoolVar(&Params, "params", false, "serve chain params info in the single call to /metrics")
	rootCmd.PersistentFlags().StringSliceVar(&Wallets, "wallets", nil, "serve info about passed wallets")
	rootCmd.PersistentFlags().StringSliceVar(&Validators, "validators", nil, "serve info about passed validators")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Err(err).Msg("Could not start application")
	}
}

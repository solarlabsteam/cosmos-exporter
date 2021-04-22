package main

import (
	"flag"
	"net/http"

	"google.golang.org/grpc"

	sdk "github.com/cosmos/cosmos-sdk/types"
	log "github.com/sirupsen/logrus"
)

var Prefix = flag.String("bech-prefix", "persistence", "Bech32 prefix for the network")
var Denom = flag.String("denom", "uxprt", "Cosmos coin denom")
var ListenAddress = flag.String("listen-address", ":9300", "The address this exporter would listen on")
var NodeAddress = flag.String("node", "localhost:9090", "RPC node address")

func main() {
	flag.Parse()

	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(*Prefix, *Prefix+"pub")
	config.SetBech32PrefixForValidator(*Prefix+"valoper", *Prefix+"valoperpub")
	config.SetBech32PrefixForConsensusNode(*Prefix+"valcons", *Prefix+"valconspub")
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

	log.Info("Listening on ", *ListenAddress)
	err = http.ListenAndServe(*ListenAddress, nil)
	if err != nil {
		log.Fatal("Could not start application at ", *ListenAddress, ", got error: ", err)
		panic(err)
	}
}

module main

go 1.16

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

require (
	github.com/cosmos/cosmos-sdk v0.46.6
	github.com/google/uuid v1.3.0
	github.com/prometheus/client_golang v1.12.2
	github.com/rs/zerolog v1.27.0
	github.com/spf13/cobra v1.6.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.13.0
	github.com/tendermint/tendermint v0.34.23
	google.golang.org/grpc v1.50.1
)

module main

go 1.16

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

replace google.golang.org/grpc => google.golang.org/grpc v1.33.2

require (
	github.com/cosmos/cosmos-sdk v0.42.4
	github.com/google/uuid v1.2.0
	github.com/prometheus/client_golang v1.8.0
	github.com/rs/zerolog v1.20.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/tendermint v0.34.9
	google.golang.org/grpc v1.35.0
)

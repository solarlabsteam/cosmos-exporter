module main

go 1.16

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

require (
	github.com/cosmos/cosmos-sdk v0.42.4
	github.com/prometheus/client_golang v1.8.0
	github.com/sirupsen/logrus v1.6.0
	google.golang.org/grpc v1.35.0
)

module github.com/flokiorg/grpc-miner

go 1.23.4

require (
	github.com/flokiorg/go-flokicoin v1.0.0
	github.com/jessevdk/go-flags v1.6.1
	github.com/rs/zerolog v1.33.0
	google.golang.org/grpc v1.71.0
	google.golang.org/protobuf v1.36.5
)

require (
	github.com/decred/dcrd/crypto/blake256 v1.1.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.3.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/net v0.37.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto v0.0.0-20250313205543-e70fdf4c4cb4 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250227231956-55c901821b1e // indirect
)

replace github.com/flokiorg/go-flokicoin => ../go-flokicoin

replace github.com/flokiorg/flokicoin-neutrino => ../flokicoin-neutrino

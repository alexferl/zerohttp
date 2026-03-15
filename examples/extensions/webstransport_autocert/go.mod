module github.com/alexferl/zerohttp/examples/webtransport_autocert

go 1.25.0

require (
	github.com/alexferl/zerohttp v0.0.0
	github.com/quic-go/quic-go v0.59.0
	github.com/quic-go/webtransport-go v0.10.0
	golang.org/x/crypto v0.49.0
)

require (
	github.com/dunglas/httpsfv v1.1.0 // indirect
	github.com/quic-go/qpack v0.6.0 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
)

replace github.com/alexferl/zerohttp => ../../..

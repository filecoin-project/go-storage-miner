module github.com/filecoin-project/go-storage-miner

go 1.13

require (
	github.com/filecoin-project/go-address v0.0.2-0.20200218010043-eb9bb40ed5be
	github.com/filecoin-project/go-cbor-util v0.0.0-20191219014500-08c40a1e63a2
	github.com/filecoin-project/go-fil-commcid v0.0.0-20200208005934-2b8bd03caca5
	github.com/filecoin-project/go-padreader v0.0.0-20200210211231-548257017ca6
	github.com/filecoin-project/go-sectorbuilder v0.0.2-0.20200306003553-541b1e74104d
	github.com/filecoin-project/go-statemachine v0.0.0-20200226041606-2074af6d51d9
	github.com/filecoin-project/specs-actors v0.0.0-20200226200336-94c9b92b2775
	github.com/hashicorp/go-multierror v1.0.0
	github.com/ipfs/go-cid v0.0.5
	github.com/ipfs/go-datastore v0.1.1
	github.com/ipfs/go-ipld-cbor v0.0.4
	github.com/ipfs/go-log v1.0.1
	github.com/ipfs/go-log/v2 v2.0.2
	github.com/multiformats/go-multihash v0.0.13
	github.com/stretchr/testify v1.4.0
	github.com/whyrusleeping/cbor-gen v0.0.0-20200206220010-03c9665e2a66
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543
)

replace github.com/golangci/golangci-lint => github.com/golangci/golangci-lint v1.18.0

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi

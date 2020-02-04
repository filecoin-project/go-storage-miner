module github.com/filecoin-project/go-storage-miner

go 1.13

require (
	github.com/filecoin-project/filecoin-ffi v0.0.0-20191219131535-bb699517a590
	github.com/filecoin-project/go-address v0.0.0-20200107215422-da8eea2842b5
	github.com/filecoin-project/go-cbor-util v0.0.0-20191219014500-08c40a1e63a2
	github.com/filecoin-project/go-padreader v0.0.0-20200129213049-ea73e2aaabd1
	github.com/filecoin-project/go-sectorbuilder v0.0.2-0.20200131010043-6b57024f839c
	github.com/filecoin-project/go-statemachine v0.0.0-20200129214539-c78c5f7e9f9c
	github.com/hashicorp/go-multierror v1.0.0
	github.com/ipfs/go-cid v0.0.4
	github.com/ipfs/go-datastore v0.1.1
	github.com/ipfs/go-ipld-cbor v0.0.3
	github.com/ipfs/go-log v1.0.1
	github.com/ipfs/go-log/v2 v2.0.2
	github.com/multiformats/go-multihash v0.0.10
	github.com/stretchr/testify v1.4.0
	github.com/whyrusleeping/cbor-gen v0.0.0-20200123233031-1cdf64d27158
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543
)

replace github.com/golangci/golangci-lint => github.com/golangci/golangci-lint v1.18.0

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi

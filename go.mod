module github.com/filecoin-project/go-storage-miner

go 1.13

require (
	github.com/filecoin-project/filecoin-ffi v0.0.0-20191219131535-bb699517a590
	github.com/filecoin-project/go-address v0.0.0-20200107215422-da8eea2842b5
	github.com/filecoin-project/go-cbor-util v0.0.0-20191219014500-08c40a1e63a2
	github.com/filecoin-project/go-padreader v0.0.0-20200210211231-548257017ca6
	github.com/filecoin-project/go-sectorbuilder v0.0.2-0.20200114015900-4103afa82689
	github.com/filecoin-project/go-statestore v0.1.0
	github.com/filecoin-project/specs-actors v0.0.0-20200210130641-2d1fbd8672cf
	github.com/ipfs/go-cid v0.0.4
	github.com/ipfs/go-datastore v0.1.1
	github.com/ipfs/go-ipld-cbor v0.0.4
	github.com/ipfs/go-log v1.0.0
	github.com/multiformats/go-multihash v0.0.10
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.4.0
	github.com/whyrusleeping/cbor-gen v0.0.0-20200206220010-03c9665e2a66
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543
	gotest.tools v2.2.0+incompatible
)

replace github.com/golangci/golangci-lint => github.com/golangci/golangci-lint v1.18.0

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi

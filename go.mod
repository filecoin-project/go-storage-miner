module github.com/filecoin-project/go-storage-mining

go 1.13

require (
	github.com/filecoin-project/go-sectorbuilder v0.0.0-20191220220745-2216fe5dabfe
	github.com/filecoin-project/lotus v0.1.5
	github.com/ipfs/go-block-format v0.0.2
	github.com/ipfs/go-cid v0.0.4
	github.com/ipfs/go-datastore v0.1.1
	github.com/ipfs/go-ipfs-blockstore v0.1.1
	github.com/ipfs/go-ipfs-ds-help v0.0.1
	github.com/ipfs/go-ipfs-files v0.0.4
	github.com/ipfs/go-ipld-format v0.0.2
	github.com/ipfs/go-log v1.0.0
	github.com/ipfs/go-unixfs v0.2.2-0.20190827150610-868af2e9e5cb
	github.com/libp2p/go-libp2p-core v0.2.4
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.4.0
	github.com/whyrusleeping/cbor-gen v0.0.0-20191216205031-b047b6acb3c0
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543
	gotest.tools v2.2.0+incompatible
)

replace github.com/golangci/golangci-lint => github.com/golangci/golangci-lint v1.18.0

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi

module github.com/filecoin-project/go-storage-miner

go 1.13

require (
	github.com/filecoin-project/filecoin-ffi v0.0.0-20191221090835-c7bbef445934
	github.com/filecoin-project/go-address v0.0.0-20200107215422-da8eea2842b5
	github.com/filecoin-project/go-cbor-util v0.0.0-20191219014500-08c40a1e63a2
	github.com/filecoin-project/go-sectorbuilder v0.0.2-0.20200114015900-4103afa82689
	github.com/filecoin-project/go-statestore v0.0.0-20200102200712-1f63c701c1e5
	github.com/golang/groupcache v0.0.0-20191227052852-215e87163ea7 // indirect
	github.com/google/go-cmp v0.3.1 // indirect
	github.com/ipfs/go-cid v0.0.4
	github.com/ipfs/go-datastore v0.3.1
	github.com/ipfs/go-ipld-cbor v0.0.3
	github.com/ipfs/go-log v1.0.1
	github.com/ipfs/go-log/v2 v2.0.2 // indirect
	github.com/multiformats/go-multihash v0.0.10
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.4.0
	github.com/whyrusleeping/cbor-gen v0.0.0-20200108194024-08b3aa60ddbb
	go.uber.org/atomic v1.5.1 // indirect
	go.uber.org/zap v1.13.0 // indirect
	golang.org/x/crypto v0.0.0-20200109152110-61a87790db17 // indirect
	golang.org/x/lint v0.0.0-20191125180803-fdd1cda4f05f // indirect
	golang.org/x/sys v0.0.0-20200113162924-86b910548bc1 // indirect
	golang.org/x/tools v0.0.0-20200114191411-189207f339b7 // indirect
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543
	gotest.tools v2.2.0+incompatible
)

replace github.com/golangci/golangci-lint => github.com/golangci/golangci-lint v1.18.0

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi

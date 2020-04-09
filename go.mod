module github.com/filecoin-project/go-storage-miner

go 1.13

require (
	github.com/filecoin-project/go-address v0.0.2-0.20200218010043-eb9bb40ed5be
	github.com/filecoin-project/sector-storage v0.0.0-20200406195014-a6d093838576
	github.com/filecoin-project/specs-actors v0.0.0-20200324235424-aef9b20a9fb1
	github.com/filecoin-project/storage-fsm v0.0.0-20200408153957-1c356922353f
	github.com/ipfs/go-datastore v0.4.4
	github.com/ipfs/go-log v1.0.3
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543
)

replace github.com/golangci/golangci-lint => github.com/golangci/golangci-lint v1.18.0

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi

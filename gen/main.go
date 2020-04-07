package main

import (
	"fmt"
	"os"

	gen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/go-storage-miner/apis/node"
	sealing "github.com/filecoin-project/storage-fsm"
)

func main() {
	err := gen.WriteTupleEncodersToFile("./apis/node/cbor_gen.go", "node",
		node.DealInfo{},
		node.DealSchedule{},
		node.PieceWithDealInfo{},
		node.SealTicket{},
		node.SealSeed{},
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = gen.WriteMapEncodersToFile("./sealing/cbor_gen.go", "sealing",
		sealing.SectorInfo{},
		sealing.Log{},
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

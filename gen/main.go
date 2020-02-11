package main

import (
	"fmt"
	"os"

	gen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/go-storage-miner/apis/node"
	"github.com/filecoin-project/go-storage-miner/sealing"
)

func main() {
	err := gen.WriteMapEncodersToFile("./apis/node/cbor_gen.go", "node",
		node.Piece{},
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

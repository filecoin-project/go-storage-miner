package main

import (
	"fmt"
	"os"

	"github.com/filecoin-project/go-storage-miner"
	gen "github.com/whyrusleeping/cbor-gen"
)

func main() {
	err := gen.WriteMapEncodersToFile("./cbor_gen.go", "storage",
		storage.Piece{},
		storage.SealTicket{},
		storage.SealSeed{},
		storage.SectorInfo{},
		storage.Log{},
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

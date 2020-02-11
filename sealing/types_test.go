package sealing

import (
	"bytes"
	"testing"

	"github.com/filecoin-project/go-storage-miner/apis/node"

	cborutil "github.com/filecoin-project/go-cbor-util"
	commcid "github.com/filecoin-project/go-fil-commcid"
	"github.com/stretchr/testify/assert"
)

func TestSectorInfoSerialization(t *testing.T) {
	var commP [32]byte

	si := &SectorInfo{
		State:     123,
		SectorNum: 234,
		Nonce:     345,
		Pieces: []node.Piece{{
			DealID:   1234,
			Size:     5,
			PieceCID: commcid.PieceCommitmentV1ToCID(commP[:]),
		}},
		CommD: []byte{32, 4},
		CommR: nil,
		Proof: nil,
		Ticket: node.SealTicket{
			BlockHeight: 345,
			TicketBytes: []byte{87, 78, 7, 87},
		},
		PreCommitMessage: nil,
		Seed:             node.SealSeed{},
		CommitMessage:    nil,
		FaultReportMsg:   nil,
		LastErr:          "hi",
	}

	b, err := cborutil.Dump(si)
	if err != nil {
		t.Fatal(err)
	}

	var si2 SectorInfo
	if err := cborutil.ReadCborRPC(bytes.NewReader(b), &si); err != nil {
		return
	}

	assert.Equal(t, si.State, si2.State)
	assert.Equal(t, si.Nonce, si2.Nonce)
	assert.Equal(t, si.SectorNum, si2.SectorNum)

	assert.Equal(t, si.Pieces, si2.Pieces)
	assert.Equal(t, si.CommD, si2.CommD)
	assert.Equal(t, si.Ticket, si2.Ticket)

	assert.Equal(t, si, si2)

}

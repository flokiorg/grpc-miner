// Copyright (c) 2024 The Flokicoin developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

//go:build darwin
// +build darwin

package mining

import (
	"bytes"
	"context"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/flokiorg/go-flokicoin/wire"
	"github.com/flokiorg/grpc-miner/mining/algo"
	"github.com/flokiorg/grpc-miner/mining/pb"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
}

type clientMock struct {
}

func (cs *clientMock) SubmitNonce(ctx context.Context, validBlock *pb.CandidateBlock, nonce uint32) (*pb.AckBlockSubmited, error) {

	buff := bytes.NewBuffer(nil)

	block := &wire.MsgBlock{}
	if err := block.Deserialize(buff); err != nil {
		return nil, err
	}

	block.Header.Nonce = uint32(nonce)

	buff.Reset()

	if err := block.Header.Serialize(buff); err != nil {
		return nil, err
	}

	log.Debug().Msg("block submited")

	return &pb.AckBlockSubmited{
		Header: hex.EncodeToString(buff.Bytes()),
	}, nil

}

func TestMiner(t *testing.T) {

	a, err := algo.Parse("sha256")
	if err != nil {
		t.Fatal(err)
	}
	block := &pb.CandidateBlock{
		Bits:         "1702c4e4",
		Header:       "0000e020b4565882aae2f7f6e1fd2b0f2f5501e9f0c7704705fe0100000000000000000002311949e9666728866d73868fccb205dd1fc6e577cfbfaa45ac36936b196c8f9e133567e4c402177fb13bbd",
		Height:       9999,
		Merkleroot:   "----",
		Amount:       99,
		Transactions: 10,
		Block:        []byte{},
	}

	request := &pb.CandidateRequest{
		Xpub: "xpubxxx",
	}

	miner := NewMiner(a, request, "localhost:9900", time.Second*60, 1, log.Logger, true)
	miner.processCandidate(context.Background(), &clientMock{}, block)
}

// Copyright (c) 2024 The Flokicoin developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package mining

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/flokiorg/go-flokicoin/wire"
	"github.com/flokiorg/grpc-miner/common"
	. "github.com/flokiorg/grpc-miner/mining/algo/common"

	"github.com/flokiorg/grpc-miner/mining/algo"
	"github.com/flokiorg/grpc-miner/mining/pb"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
}

type clientMockSuccess struct {
}

func (cs *clientMockSuccess) SubmitNonce(ctx context.Context, validBlock *pb.CandidateBlock, nonce uint32, maxRetries int, maxBackoffSeconds float64) (*pb.AckBlockSubmited, error) {

	buff := bytes.NewBuffer(validBlock.Block)
	block := &wire.MsgBlock{}
	if err := block.Deserialize(buff); err != nil {
		return nil, fmt.Errorf("failed deserializing buff: %v", err)
	}

	block.Header.Nonce = uint32(nonce)

	buff.Reset()
	if err := block.Header.Serialize(buff); err != nil {
		return nil, fmt.Errorf("failed serializing buff: %v", err)
	}

	return &pb.AckBlockSubmited{
		Header: hex.EncodeToString(buff.Bytes()),
	}, nil

}

type clientMockFail struct {
}

func (cs *clientMockFail) SubmitNonce(ctx context.Context, validBlock *pb.CandidateBlock, nonce uint32, maxRetries int, maxBackoffSeconds float64) (*pb.AckBlockSubmited, error) {
	return nil, errors.New("unknown error")
}

func createCandidateBlock(t *testing.T, diff string) *pb.CandidateBlock {
	header := "0000e020b4565882aae2f7f6e1fd2b0f2f5501e9f0c7704705fe0100000000000000000002311949e9666728866d73868fccb205dd1fc6e577cfbfaa45ac36936b196c8f9e133567e4c402177fb13bbd"

	headerBytes, err := hex.DecodeString(header)
	if err != nil {
		t.Fatal(err)
	}

	var blockHeader wire.BlockHeader
	if err := blockHeader.Deserialize(bytes.NewReader(headerBytes)); err != nil {
		t.Fatal(err)
	}

	buff := bytes.NewBuffer(nil)
	msgBlock := wire.NewMsgBlock(&blockHeader)
	msgBlock.Serialize(buff)

	return &pb.CandidateBlock{
		Bits:         diff,
		Header:       header,
		Height:       999999,
		Merkleroot:   "----",
		Amount:       99,
		Transactions: 10,
		Block:        buff.Bytes(),
	}
}

func TestMiningBlocks(t *testing.T) {

	hashAlgo, err := algo.Parse("scrypt_cpu")
	if err != nil {
		t.Fatal(err)
	}

	block := createCandidateBlock(t, "207fffff")

	request := &pb.CandidateRequest{
		Xpub: "xpubxxx",
	}

	cfg := &common.Config{
		PoolServer:       "localhost:9900",
		SlowDownDuration: time.Second * 60,
		Threads:          uint8(runtime.NumCPU()),
		MineOnce:         true,
	}

	tests := []struct {
		name           string
		targetBlocks   int
		expectedBlocks uint32
		client         ClientService
	}{
		{"success solo mining", 2, 2, &clientMockSuccess{}},
		{"fail solo mining", 2, 0, &clientMockFail{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			miner := NewMiner(cfg, hashAlgo, request, log.Logger)
			for i := 0; i < test.targetBlocks; i++ {
				miner.wg.Add(1)
				miner.processCandidate(context.Background(), test.client, block)
			}

			if miner.acceptedBlocks != test.expectedBlocks {
				t.Fatalf("unexpected block mining result, want=%d got=%d", test.expectedBlocks, miner.acceptedBlocks)
			}
		})
	}
}

func TestStartMining(t *testing.T) {

	hashAlgo, err := algo.Parse("scrypt_cpu")
	if err != nil {
		t.Fatal(err)
	}

	request := &pb.CandidateRequest{
		Xpub: "xpubxxx",
	}

	cfg := &common.Config{
		PoolServer:       "localhost:9900",
		SlowDownDuration: time.Second * 60,
		Threads:          uint8(runtime.NumCPU()),
		MineOnce:         true,
	}

	blocks := []*pb.CandidateBlock{
		createCandidateBlock(t, "1935a7f1"), // hard
		createCandidateBlock(t, "207fffff"), // easy
	}

	miner := NewMiner(cfg, hashAlgo, request, log.Logger)

	ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
	for _, block := range blocks {
		miner.start(ctx, &clientMockSuccess{}, block)
		time.Sleep(time.Second * 5)
	}

	if miner.acceptedBlocks != 1 {
		t.Fatalf("unexpected mining result, block exepcted=%d, got=%d", 1, miner.acceptedBlocks)
	}

}

func TestMineWithIssue(t *testing.T) {

	hashAlgo, err := algo.Parse("scrypt_cpu")
	if err != nil {
		t.Fatal(err)
	}

	request := &pb.CandidateRequest{
		Xpub: "xpubxxx",
	}

	cfg := &common.Config{
		PoolServer:       "localhost:9900",
		SlowDownDuration: time.Second * 60,
		Threads:          uint8(runtime.NumCPU()),
		MineOnce:         true,
	}

	blocks := []*pb.CandidateBlock{
		createCandidateBlock(t, "207fffff"), // easy
		createCandidateBlock(t, "207fffff"), // easy
	}

	miner := NewMiner(cfg, hashAlgo, request, log.Logger)

	ctx, _ := context.WithTimeout(context.Background(), time.Second*60)
	for _, block := range blocks {
		miner.start(ctx, &clientMockFail{}, block)
	}

	miner.wg.Wait()

	if miner.acceptedBlocks != 1 {
		t.Fatalf("unexpected mining result, block exepcted=%d, got=%d", 1, miner.acceptedBlocks)
	}

}

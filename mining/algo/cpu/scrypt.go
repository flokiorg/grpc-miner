// Copyright (c) 2024 The Flokicoin developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package cpu

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"math/big"

	"github.com/flokiorg/grpc-miner/hash/scrypt"
	. "github.com/flokiorg/grpc-miner/mining/algo/common"
	"github.com/flokiorg/grpc-miner/mining/pb"
	"github.com/flokiorg/grpc-miner/utils"
	"github.com/rs/zerolog/log"
)

const (
	NUM_ITERATIONS = 1000
)

type SdtScrypt struct{}

func NewSdtScrypt() *SdtScrypt {
	return &SdtScrypt{}
}

func (fs *SdtScrypt) Mine(ctx context.Context, stats *Stats, block *pb.CandidateBlock, nonceRange utils.MinMax, tid uint8) (string, uint32, error) {

	_, targetDifficulty := utils.CalcDifficulty(block.Bits) // targetLenDifficulty

	blockBytes, _ := hex.DecodeString(block.Header[:BLOCK_NONCELESS_LENGTH])
	buffer := bytes.NewBuffer(nil)
	var nonce uint32 = nonceRange.Min
	var currIterations uint32 = 0

	for nonce <= nonceRange.Max {

		binary.Write(buffer, binary.LittleEndian, blockBytes)
		binary.Write(buffer, binary.LittleEndian, nonce)

		blockhashBytes, err := scrypt.Key(buffer.Bytes(), buffer.Bytes(), 1024, 1, 1, 32)
		if err != nil {
			log.Fatal().Err(err).Msg("mining")
		}

		utils.ReverseBytes(blockhashBytes)

		hashNum := &big.Int{}
		hashNum.SetBytes(blockhashBytes)

		if hashNum.Cmp(targetDifficulty) < 0 {
			return hex.EncodeToString(blockhashBytes), nonce, nil
		}

		buffer.Reset()
		nonce++
		currIterations++

		select {
		case <-ctx.Done():
			return "", 0, ErrMiningCancelled

		default:
			if currIterations%NUM_ITERATIONS == 0 {
				stats.TotalHashes.Add(uint64(NUM_ITERATIONS))
				stats.Iterations.Add(1)
				currIterations = 0
			}
		}
	}

	return "", 0, ErrMiningCompleted

}

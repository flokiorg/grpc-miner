// Copyright (c) 2024 The Flokicoin developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package algo

import (
	"context"
	"errors"
	"strings"

	. "github.com/flokiorg/grpc-miner/mining/algo/common"
	"github.com/flokiorg/grpc-miner/mining/algo/cpu"
	"github.com/flokiorg/grpc-miner/mining/pb"
	"github.com/flokiorg/grpc-miner/utils"
)

type MinerAlgo interface {
	Mine(ctx context.Context, stats *Stats, block *pb.CandidateBlock, nonceRange utils.MinMax, tid uint8) (string, uint32, error)
}

type ALGO int

const (
	SHA256 ALGO = iota
	SCRYPT
)

func Parse(input string) (MinerAlgo, error) {
	switch strings.ToLower(input) {

	case "scrypt_cpu":
		return cpu.NewSdtScrypt(), nil

		// case "sha256_gpu":
		// 	return gpu.NewFastSha256(), nil

		// case "scrypt_gpu":
		// 	return gpu.NewSdtScrypt(), nil

	}

	return nil, errors.New("unsupported algo")
}

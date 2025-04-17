// Copyright (c) 2024 The Flokicoin developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package common

import (
	"context"

	"github.com/flokiorg/grpc-miner/mining/pb"
)

type ClientService interface {
	SubmitNonce(context.Context, *pb.CandidateBlock, uint32, int, float64) (*pb.AckBlockSubmited, error)
}

// Copyright (c) 2024 The Flokicoin developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package mining

import (
	"context"
	"fmt"
	"time"

	"github.com/flokiorg/grpc-miner/mining/pb"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	stream pb.CandidateStreamClient
}

func NewClient(poolserver string) (*Client, error) {
	conn, err := grpc.NewClient(poolserver, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := pb.NewHealthClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	_, err = client.Check(ctx, &pb.HealthCheckRequest{})
	if err != nil {
		return nil, fmt.Errorf("Health check failed: %v\n", err)
	}

	return &Client{
		conn:   conn,
		stream: pb.NewCandidateStreamClient(conn),
	}, nil

}

func (c *Client) Listen(ctx context.Context, cRequest *pb.CandidateRequest, blocks chan<- *pb.CandidateBlock, errors chan<- error) {

	stream, err := c.stream.Open(ctx, cRequest)
	if err != nil {
		errors <- err
		return
	}

	log.Info().Msg("listening for candidates blocks...")

	for {
		select {
		case <-ctx.Done():
			return

		default:
			input, err := stream.Recv()
			if err != nil {
				errors <- err
				return
			}

			blocks <- input
		}
	}
}

func (c *Client) SubmitNonce(ctx context.Context, candicateBlock *pb.CandidateBlock, nonce uint32) (*pb.AckBlockSubmited, error) {
	return c.stream.SubmitValidBlock(ctx, &pb.ValidBlock{Template: candicateBlock, Nonce: int64(nonce)})
}

func (c *Client) Generate(ctx context.Context, blocks int) ([]string, error) {
	res, err := c.stream.Generate(ctx, &pb.GenerateRequest{
		NumBlocks: int32(blocks),
	})
	if err != nil {
		return nil, err
	}

	return res.Blocks, nil
}

// Copyright (c) 2024 The Flokicoin developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package mining

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/flokiorg/grpc-miner/mining/pb"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn       *grpc.ClientConn
	stream     pb.CandidateStreamClient
	retryChan  chan struct{}
	retryMutex sync.Mutex
}

// NewClient initializes a new gRPC client
func NewClient(poolserver string, dialTimeout time.Duration) (*Client, error) {

	conn, err := grpc.NewClient(poolserver, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to gRPC server")
		return nil, err
	}

	client := pb.NewHealthClient(conn)

	// Health check
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()

	_, err = client.Check(ctx, &pb.HealthCheckRequest{})
	if err != nil {
		conn.Close()
		log.Error().Err(err).Msg("Health check failed, closing connection")
		return nil, fmt.Errorf("health check failed: %v", err)
	}

	log.Info().Msg("client initialized successfully")
	return &Client{
		conn:   conn,
		stream: pb.NewCandidateStreamClient(conn),
	}, nil
}

// Listen continuously listens for candidate blocks and synchronizes retries
func (c *Client) Listen(ctx context.Context, request *pb.CandidateRequest, blocks chan<- *pb.CandidateBlock) {
	var attempt int

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Listen stopped by context cancellation")
			return
		default:
		}

		log.Info().Int("attempt", attempt).Msg("Opening stream to listen for candidate blocks...")

		stream, err := c.stream.Open(ctx, request)
		if err != nil {
			log.Error().Err(err).Msg("Stream open failed, retrying...")

			// Notify `SubmitNonce` that connection is unstable
			select {
			case c.retryChan <- struct{}{}:
			default:
			}

			// Exponential backoff before retrying
			attempt++
			backoff := time.Duration(math.Min(30, math.Pow(2, float64(attempt)))) * time.Second
			log.Warn().Dur("retry_after", backoff).Msg("Retrying stream open...")
			time.Sleep(backoff)

			continue // retry opening stream
		}

		log.Info().Msg("Listening for candidate blocks...")
		attempt = 0 //  Reset retry counter on success

	loop:
		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("Stopping Listen due to context cancellation")
				return
			default:
				input, err := stream.Recv()
				if err != nil {
					log.Warn().Err(err).Msg("Stream error detected, retrying...")

					// Notify `SubmitNonce` to wait for reconnection
					select {
					case c.retryChan <- struct{}{}:
					default:
					}

					attempt++
					backoff := time.Duration(math.Min(30, math.Pow(2, float64(attempt)))) * time.Second
					log.Warn().Dur("retry_after", backoff).Msg("Retrying stream open...")
					time.Sleep(backoff)

					break loop //  Restart connection loop
				}

				log.Info().Str("block", fmt.Sprintf("%v", input.Height)).Msg("Received candidate block")
				blocks <- input
			}
		}
	}
}

// SubmitNonce waits if `Listen` is retrying, then submits the nonce
func (c *Client) SubmitNonce(ctx context.Context, block *pb.CandidateBlock, nonce uint32, maxRetries int, maxBackoffSeconds float64) (*pb.AckBlockSubmited, error) {
	var attempt int

	log.Info().
		Str("block", fmt.Sprintf("%v", block.Height)).
		Uint32("nonce", nonce).
		Msg("Submitting nonce...")

	for {
		select {
		case <-ctx.Done():
			log.Warn().Msg("block submission halted due to context cancellation")
			return nil, ctx.Err()

		case <-c.retryChan: // Wait if Listen is retrying
			log.Warn().Msg("Detected connection retry, delaying nonce submission...")
			time.Sleep(2 * time.Second) // Short delay before checking again
			continue

		default:
		}

		// Prevent multiple retry attempts
		c.retryMutex.Lock()
		resp, err := c.stream.SubmitValidBlock(ctx, &pb.ValidBlock{Template: block, Nonce: int64(nonce)})
		c.retryMutex.Unlock()

		if err == nil {
			log.Info().
				Str("block", fmt.Sprintf("%v", block.Height)).
				Uint32("nonce", nonce).
				Msg("block submitted successfully")
			return resp, nil
		}

		// Log error and retry if applicable
		attempt++
		if attempt > maxRetries {
			log.Error().
				Str("block", fmt.Sprintf("%v", block.Height)).
				Uint32("nonce", nonce).
				Int("attempts", attempt).
				Err(err).
				Msg("Failed to submit nonce after multiple attempts")
			return nil, fmt.Errorf("block submission failed after %d attempts: %v", attempt, err)
		}

		// Calculate backoff time
		backoff := time.Duration(math.Min(maxBackoffSeconds, math.Pow(2, float64(attempt)))) * time.Second
		log.Warn().
			Str("block", fmt.Sprintf("%v", block.Height)).
			Uint32("nonce", nonce).
			Int("attempts", attempt).
			Dur("retry_after", backoff).
			Err(err).
			Msg("Retrying block submission...")

		time.Sleep(backoff) // Wait before retrying
	}
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

func (c *Client) Close() {
	c.conn.Close()
}

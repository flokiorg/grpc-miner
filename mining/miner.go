// Copyright (c) 2024 The Flokicoin developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package mining

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/flokiorg/grpc-miner/hash/sha256"
	"github.com/flokiorg/grpc-miner/mining/algo"
	. "github.com/flokiorg/grpc-miner/mining/algo/common"
	"github.com/flokiorg/grpc-miner/mining/pb"
	"github.com/flokiorg/grpc-miner/utils"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Miner struct {
	slowDownDuration time.Duration
	poolserver       string
	candidateRequest *pb.CandidateRequest
	ma               algo.MinerAlgo
	maxThreads       uint8

	stats *Stats

	mu         sync.Mutex
	retryDelay time.Duration
	logger     zerolog.Logger

	mineonce bool
}

type workers struct {
	quit        chan struct{}
	wg          sync.WaitGroup
	block       *pb.CandidateBlock
	client      ClientService
	nonceRanges map[uint8]utils.MinMax
	algo        algo.MinerAlgo
	stats       *Stats

	mu            sync.Mutex
	blockSubmited bool
}

func (w *workers) run(ctx context.Context, tid uint8, logger zerolog.Logger) {
	defer func() {
		w.wg.Done()
		log.Debug().Msgf("b[%d] t[%d] 🏁 completed", w.block.Height, tid)
	}()

	nonceRange := w.nonceRanges[tid]
	log.Debug().Msgf("b[%d] t[%d] nonce.range=(%d, %d)", w.block.Height, tid, nonceRange.Min, nonceRange.Max)

	blockhash, nonce, err := w.algo.Mine(w.stats, w.block, nonceRange, tid, w.quit)
	if err != nil {
		if !errors.Is(err, ErrMiningCancelled) && !errors.Is(err, ErrMiningCompleted) {
			log.Error().Err(err).Msg("mining failed")
		}
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.blockSubmited {
		return
	}

	w.blockSubmited = true
	close(w.quit)

	logger.Info().Msgf("b[%d] t[%d] ✨ nonce:%d", w.block.Height, tid, nonce)
	logger.Info().Msgf("b[%d] t[%d] ✨ solved hash:%s", w.block.Height, tid, blockhash)

	ack, err := w.client.SubmitNonce(ctx, w.block, nonce)
	if err != nil {
		log.Error().Err(err).Msgf("b[%d] t[%d] ❌ failed submiting block.", w.block.Height, tid)
	} else {
		headerBytes, _ := hex.DecodeString(ack.Header)
		blockhash := sha256.DoubleSum256(headerBytes)
		utils.ReverseBytes(blockhash)
		logger.Info().Msgf("b[%d] t[%d] ✨ block submited", w.block.Height, tid)
		logger.Info().Msgf("b[%d] t[%d] ✨ blockhash:%x", w.block.Height, tid, blockhash)
	}

}

func NewMiner(ma algo.MinerAlgo, cRequest *pb.CandidateRequest, poolserver string, slowDownDuration time.Duration, threads uint8, logger zerolog.Logger, mineonce bool) *Miner {
	return &Miner{
		ma:               ma,
		stats:            NewStats(),
		slowDownDuration: slowDownDuration,
		poolserver:       poolserver,
		candidateRequest: cRequest,
		maxThreads:       threads,
		logger:           logger,
		mineonce:         mineonce,
	}
}

func (m *Miner) processCandidate(ctx context.Context, client ClientService, block *pb.CandidateBlock) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stats.Reset()

	lenDifficulty, _ := utils.CalcDifficulty(block.Bits)

	m.logger.Info().Msgf("🌱 new block height:%d", block.Height)
	m.logger.Info().Msgf("processing block: %d amount: %d txs: %d", block.Height, block.Amount, block.Transactions)
	m.logger.Info().Msgf("version: %d", block.Version)
	m.logger.Info().Msgf("target difficulty: %s/%d", block.Bits, lenDifficulty)
	m.logger.Info().Msgf("merkleroot: %s", block.Merkleroot)
	m.logger.Info().Msgf("address: %s", block.Address)

	workers := workers{
		quit:        make(chan struct{}),
		wg:          sync.WaitGroup{},
		block:       block,
		client:      client,
		algo:        m.ma,
		stats:       m.stats,
		nonceRanges: utils.CalculateNonceRanges(TOTAL_NONCES, START_NONCE, m.maxThreads),
	}

	for tid := uint8(0); tid < m.maxThreads; tid++ {
		workers.wg.Add(1)
		go workers.run(ctx, tid, m.logger)
	}

	go func(ctx context.Context, m *Miner) {
		startime := time.Now()

		ticker := time.NewTicker(time.Second * 1)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				m.stats.PrintProgress(block, startime, TOTAL_NONCES)

			case <-ctx.Done():
				return
			}
		}
	}(ctx, m)

	workers.wg.Wait()

	if m.mineonce {
		log.Debug().Msg("Program exiting: 'mineonce' is activated, mining completed.")
		os.Exit(0)
	}

	if workers.blockSubmited && m.slowDownDuration > 0 {
		m.logger.Info().Msgf("🚦 slow down mining for %d secs", int(m.slowDownDuration.Seconds()))
		time.Sleep(m.slowDownDuration)
	} else {
		m.logger.Info().Msgf("🚦 skipped")
	}

}

func (m *Miner) Run(ctx context.Context) {

	for {
		err := m.start(ctx)
		if err == nil {
			return
		}
		log.Error().Err(err).Msgf("failed")

		if m.retryDelay < time.Minute {
			if m.retryDelay == 0 {
				m.retryDelay = 3 * time.Second
			} else {
				m.retryDelay += 3 * time.Second
				if m.retryDelay > time.Minute {
					m.retryDelay = time.Minute
				}
			}
		}

		log.Debug().Msgf("retrying in %s", m.retryDelay)
		select {
		case <-time.After(m.retryDelay):
		case <-ctx.Done():
			log.Debug().Msg("context canceled, exiting")
			return
		}

	}
}

func (m *Miner) start(parent context.Context) error {

	client, err := NewClient(m.poolserver)
	if err != nil {
		return fmt.Errorf("Failed to establish connection to the pool server at %s", m.poolserver)
	}

	blocks := make(chan *pb.CandidateBlock)
	errors := make(chan error)
	m.retryDelay = 0

	go client.Listen(parent, m.candidateRequest, blocks, errors)

	var cancelFunc context.CancelFunc

	for {

		select {
		case block := <-blocks:
			m.logger.Info().Msgf("! block received %d", block.Height)
			if cancelFunc != nil {
				cancelFunc()
			}

			ctx, cancel := context.WithCancel(parent)
			cancelFunc = cancel

			go m.processCandidate(ctx, client, block)

		case err := <-errors:
			if cancelFunc != nil {
				cancelFunc()
			}
			return fmt.Errorf("unexpected error, %w", err)

		case <-parent.Done():
			m.logger.Info().Msgf("! parent cancelled")

		}

	}
}

func (m *Miner) Generate(ctx context.Context, numBlocks int) {

	client, err := NewClient(m.poolserver)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to establish connection to the pool server at %s", m.poolserver)
	}

	blocks, err := client.Generate(ctx, numBlocks)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to establish connection to the pool server at %s", m.poolserver)
	}

	fmt.Println("Generated Blocks:")
	for i, block := range blocks {
		fmt.Printf(" #%d:\t%s\n", i+1, block)
	}

}

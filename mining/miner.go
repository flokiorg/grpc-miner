// Copyright (c) 2024 The Flokicoin developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package mining

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flokiorg/grpc-miner/common"
	"github.com/flokiorg/grpc-miner/hash/sha256"
	"github.com/flokiorg/grpc-miner/mining/algo"
	. "github.com/flokiorg/grpc-miner/mining/algo/common"
	"github.com/flokiorg/grpc-miner/mining/pb"
	"github.com/flokiorg/grpc-miner/utils"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Miner struct {
	candidateRequest *pb.CandidateRequest
	ma               algo.MinerAlgo
	cfg              *common.Config
	stats            *Stats
	mu               sync.Mutex
	logger           zerolog.Logger

	acceptedBlocks uint32
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

type workers struct {
	wg          sync.WaitGroup
	block       *pb.CandidateBlock
	nonceRanges map[uint8]utils.MinMax
	algo        algo.MinerAlgo
	stats       *Stats
	mu          sync.Mutex

	cancel    context.CancelFunc
	blockhash string
	nonce     uint32
}

func (w *workers) run(ctx context.Context, tid uint8, logger zerolog.Logger) {
	defer func() {
		w.wg.Done()
		log.Debug().Msgf("b[%d] t[%d] 🏁 completed", w.block.Height, tid)
	}()

	nonceRange := w.nonceRanges[tid]
	log.Debug().Msgf("b[%d] t[%d] nonce.range=(%d, %d)", w.block.Height, tid, nonceRange.Min, nonceRange.Max)

	blockhash, nonce, err := w.algo.Mine(ctx, w.stats, w.block, nonceRange, tid)
	if err == nil {
		w.mu.Lock()
		w.blockhash = blockhash
		w.nonce = nonce
		w.mu.Unlock()
		w.cancel()
		return
	}
	if !errors.Is(err, ErrMiningCancelled) && !errors.Is(err, ErrMiningCompleted) {
		logger.Error().Err(err).Msg("mining failed")
	}
}

func NewMiner(cfg *common.Config, ma algo.MinerAlgo, request *pb.CandidateRequest, logger zerolog.Logger) *Miner {
	return &Miner{
		cfg:              cfg,
		ma:               ma,
		stats:            NewStats(),
		logger:           logger,
		candidateRequest: request,
	}
}

func (m *Miner) processCandidate(parent context.Context, client ClientService, block *pb.CandidateBlock) {
	defer m.wg.Done()

	m.stats.Reset()

	lenDifficulty, _ := utils.CalcDifficulty(block.Bits)

	m.logger.Info().Msgf("🌱 new block height:%d", block.Height)
	m.logger.Info().Msgf("processing block: %d amount: %d txs: %d", block.Height, block.Amount, block.Transactions)
	m.logger.Info().Msgf("version: %d", block.Version)
	m.logger.Info().Msgf("target difficulty: %s/%d", block.Bits, lenDifficulty)
	m.logger.Info().Msgf("merkleroot: %s", block.Merkleroot)
	m.logger.Info().Msgf("address: %s", block.Address)

	ctx, cancel := context.WithCancel(parent)

	workers := workers{
		cancel:      cancel,
		wg:          sync.WaitGroup{},
		block:       block,
		algo:        m.ma,
		stats:       m.stats,
		nonceRanges: utils.CalculateNonceRanges(TOTAL_NONCES, START_NONCE, m.cfg.Threads),
	}

	for tid := uint8(0); tid < m.cfg.Threads; tid++ {
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

	if len(workers.blockhash) > 0 {
		m.logger.Info().Msgf("b[%d] ✨ nonce:%d", block.Height, workers.nonce)
		m.logger.Info().Msgf("b[%d] ✨ solved hash:%s", block.Height, workers.blockhash)

		ack, err := client.SubmitNonce(parent, block, workers.nonce, m.cfg.MaxRetries, m.cfg.MaxBackoffSeconds)
		if err != nil {
			m.logger.Error().Err(err).Msgf("b[%d] ❌ failed submiting block.", block.Height)
		} else {
			headerBytes, _ := hex.DecodeString(ack.Header)
			blockhash := sha256.DoubleSum256(headerBytes)
			utils.ReverseBytes(blockhash)
			m.logger.Info().Msgf("b[%d] ✨ block submited", block.Height)
			m.logger.Info().Msgf("b[%d] ✨ blockhash:%x", block.Height, blockhash)

			atomic.AddUint32(&m.acceptedBlocks, 1)

			if !m.cfg.MineOnce && m.cfg.SlowDownDuration > 0 {
				m.logger.Info().Msgf("🚦 slow down mining for %d secs", int(m.cfg.SlowDownDuration.Seconds()))
				time.Sleep(m.cfg.SlowDownDuration)
			}
		}
	}

}

func (m *Miner) start(parent context.Context, client ClientService, block *pb.CandidateBlock) {
	select {
	case <-parent.Done():
		log.Info().Msg("failed to start, context cancelled")
		return
	default:

	}

	m.mu.Lock()
	ctx, cancel := context.WithCancel(parent)
	m.cancel = cancel
	m.mu.Unlock()

	m.wg.Add(1)
	go m.processCandidate(ctx, client, block)
}

func (m *Miner) stop() {

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
		m.wg.Wait()
	}

}

func (m *Miner) Run(ctx context.Context) {

	client, err := NewClient(m.cfg.PoolServer, m.cfg.PoolTimeout)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to establish connection to the pool server at %s", m.cfg.PoolServer)
	}
	defer client.Close()

	blocks := make(chan *pb.CandidateBlock)

	go client.Listen(ctx, m.candidateRequest, blocks)

	var previousBlockHeight int64
	for {
		select {
		case block := <-blocks:
			if m.cfg.MineOnce && atomic.LoadUint32(&m.acceptedBlocks) > 0 {
				return
			}

			m.stop()

			if previousBlockHeight != 0 && block.Height > previousBlockHeight && m.cfg.BlockSiesta > 0 {
				m.logger.Info().Msgf("😴 Taking a siesta for %d secs", int(m.cfg.BlockSiesta.Seconds()))
				time.Sleep(m.cfg.BlockSiesta)
			}

			previousBlockHeight = block.Height

			m.start(ctx, client, block)

		case <-ctx.Done():
			return
		}
	}
}

func (m *Miner) Generate(ctx context.Context, numBlocks int) {

	client, err := NewClient(m.cfg.PoolServer, m.cfg.PoolTimeout)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to establish connection to the pool server at %s", m.cfg.PoolServer)
	}
	defer client.Close()

	blocks, err := client.Generate(ctx, numBlocks)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to generate blocks")
	}

	fmt.Println("Generated Blocks:")
	for i, block := range blocks {
		fmt.Printf(" #%d:\t%s\n", i+1, block)
	}

}

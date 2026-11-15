// Copyright (c) 2024 The Flokicoin developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package common

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flokiorg/grpc-miner/mining/pb"
	"github.com/rs/zerolog/log"
)

type Stats struct {
	Iterations  atomic.Uint64
	TotalHashes atomic.Uint64

	zeros     map[uint8]int
	zerosLock sync.Mutex

	lastTotalHashes uint64
}

func NewStats() *Stats {
	return &Stats{
		zeros: make(map[uint8]int),
	}
}

func (s *Stats) IncZeros(zs map[uint8]int) {
	s.zerosLock.Lock()
	defer s.zerosLock.Unlock()

	for z, c := range zs {
		s.zeros[z] += c
	}
}

func (s *Stats) Reset() {
	s.Iterations.Store(0)
	s.TotalHashes.Store(0)

	s.zerosLock.Lock()
	s.zeros = make(map[uint8]int)
	s.zerosLock.Unlock()

	s.lastTotalHashes = 0
}

func (s *Stats) PrintZeros() {
	if len(s.zeros) == 0 {
		return
	}

	output := []string{}

	s.zerosLock.Lock()
	for z, c := range s.zeros {
		output = append(output, fmt.Sprintf("	[%d]: %d", z, c))
	}
	s.zerosLock.Unlock()

	log.Debug().Msgf("[stats(%d)]: \n%s", len(output), strings.Join(output, "\n"))
}

func (s *Stats) PrintProgress(block *pb.CandidateBlock, startime time.Time, totalNonces uint32) {

	cptIterations := s.Iterations.Load()
	totalHashes := s.TotalHashes.Load()
	hashes := totalHashes - s.lastTotalHashes
	s.lastTotalHashes = totalHashes
	megahashes_per_second := (float64(totalHashes) / time.Since(startime).Seconds()) / 1_000_000
	progress := (float64(totalHashes) / float64(totalNonces)) * 100

	log.Debug().Msgf("b[%d] %d iterations | hashrate: %.5f MH/s | hashes: %d | total: %d | progress: %.2f%%", block.Height, cptIterations, megahashes_per_second, hashes, totalHashes, progress)
}

// Copyright (c) 2024 The Flokicoin developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package utils

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/flokiorg/go-flokicoin/blockchain"
)

func ReverseBytes(b []byte) {
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
}

func CalcDifficulty(bits string) (uint8, *big.Int) {
	bitsBytes, _ := hex.DecodeString(bits)
	ReverseBytes(bitsBytes)
	nbits := binary.LittleEndian.Uint32(bitsBytes)
	target := blockchain.CompactToBig(nbits)
	return uint8(len(target.String())), target
}

type MinMax struct {
	Min uint32
	Max uint32
}

func CalculateNonceRanges(totalNonces, startNonce uint32, numThreads uint8) map[uint8]MinMax {
	nonceRanges := make(map[uint8]MinMax)
	adjustedRange := totalNonces - startNonce
	noncesPerThread := adjustedRange / uint32(numThreads)

	for i := uint8(0); i < numThreads; i++ {
		min := startNonce + uint32(i)*noncesPerThread
		var max uint32
		if i == numThreads-1 {
			max = totalNonces
		} else {
			max = min + noncesPerThread - 1
		}
		nonceRanges[i] = MinMax{Min: min, Max: max}
	}
	return nonceRanges
}

func BytesToUint32(b []byte) ([]uint32, error) {
	if len(b) == 0 {
		return nil, fmt.Errorf("input byte slice is empty")
	}

	if len(b)%4 != 0 {
		return nil, fmt.Errorf("byte slice length must be a multiple of 4")
	}

	uint32Array := make([]uint32, len(b)/4)

	err := binary.Read(bytes.NewReader(b), binary.BigEndian, &uint32Array)
	if err != nil {
		return nil, fmt.Errorf("failed to convert bytes to uint32: %w", err)
	}

	return uint32Array, nil
}

func Uint32ToBytes(input []uint32) []byte {
	output := make([]uint8, len(input)*4)

	for i, value := range input {
		binary.BigEndian.PutUint32(output[i*4:i*4+4], value)
	}

	return output
}

func HexStringToWords(hexStr string) ([]uint32, error) {
	if len(hexStr)%8 != 0 {
		return nil, fmt.Errorf("hex string length must be a multiple of 8")
	}

	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, fmt.Errorf("invalid hex string: %v", err)
	}

	words := make([]uint32, len(bytes)/4)

	for i := 0; i < len(bytes); i += 4 {
		words[i/4] = (uint32(bytes[i]) << 24) | // Most significant byte
			(uint32(bytes[i+1]) << 16) |
			(uint32(bytes[i+2]) << 8) |
			(uint32(bytes[i+3])) // Least significant byte
	}

	return words, nil
}

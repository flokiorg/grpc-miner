// Copyright (c) 2024 The Flokicoin developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package cpu

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/flokiorg/grpc-miner/hash/scrypt"
	. "github.com/flokiorg/grpc-miner/mining/algo/common"
	"github.com/flokiorg/grpc-miner/utils"
)

func TestMine(t *testing.T) {
	header := "00000020d7d2fc3301d304edfcffeafd0d41d0bd507d4622bc464fd92deddc94c9cfd9b89c1b8cb9fc61ffbdaa88602b2fce770bc9fcdc296ba47f522b5d9d829b887833406d7167e2554219"

	headerBytes, _ := hex.DecodeString(header[:BLOCK_NONCELESS_LENGTH])
	buffer := bytes.NewBuffer(headerBytes)

	var nonce uint32 = 1124238675
	binary.Write(buffer, binary.LittleEndian, nonce)

	res, _ := scrypt.Key(buffer.Bytes(), buffer.Bytes(), 1024, 1, 1, 32)
	t.Logf("res: %x", res)

	buffer.Reset()
	binary.Write(buffer, binary.LittleEndian, headerBytes)
	nonce++
	binary.Write(buffer, binary.LittleEndian, nonce)

	res, _ = scrypt.Key(buffer.Bytes(), buffer.Bytes(), 1024, 1, 1, 32)
	t.Logf("res: %x", res)
}

func TestHash(t *testing.T) {
	header := "00000020d7d2fc3301d304edfcffeafd0d41d0bd507d4622bc464fd92deddc94c9cfd9b89c1b8cb9fc61ffbdaa88602b2fce770bc9fcdc296ba47f522b5d9d829b887833406d7167e2554219"

	headerBytes, _ := hex.DecodeString(header[:BLOCK_NONCELESS_LENGTH])
	buffer := bytes.NewBuffer(headerBytes)

	var nonce uint32 = 1124238675
	binary.Write(buffer, binary.LittleEndian, nonce)

	blockhash, _ := scrypt.Key(buffer.Bytes(), buffer.Bytes(), 1024, 1, 1, 32)

	utils.ReverseBytes(blockhash)

	hashBig := &big.Int{}
	hashBig.SetBytes(blockhash)

	t.Logf("blockhash: %x", blockhash)
	t.Logf("nonce: %x", nonce)
	t.Logf("hashBig: %s", hashBig.String())
}

// Copyright (c) 2024 The Flokicoin developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package common

import (
	"math"
	"runtime"
)

var (
	DefaultThreadsMax = uint8(runtime.NumCPU()) // 5
)

const (
	BLOCK_NONCELESS_LENGTH = 152
	BLOCK_LENGTH           = 80

	SHA256_HASH_SIZE = 32

	TOTAL_NONCES uint32 = math.MaxUint32 // 4_294_967_295

	START_NONCE uint32 = 0 // 170000000 // 1_550_000_000

)

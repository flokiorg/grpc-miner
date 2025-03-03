// Copyright (c) 2024 The Flokicoin developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package common

import "errors"

var (
	ErrMiningCancelled = errors.New("mining canceled")
	ErrMiningCompleted = errors.New("mining completed")
)

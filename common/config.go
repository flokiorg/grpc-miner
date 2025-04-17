// Copyright (c) 2024 The Flokicoin developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package common

import "time"

const (
	MaxCoinbaseScriptSize       = 50
	DefaultCBSBoundaryBytesSize = 5
)

type Config struct {
	ConfigFile        string        `short:"c" long:"config" description:"Path to configuration file"`
	Algo              string        `short:"a" long:"algo" description:"Algorithm to use for mining (scrypt_cpu, scrypt_gpu, sha256_cpu, sha256_gpu)"`
	Threads           uint8         `short:"t" long:"threads" description:"Number of threads to use (default: all available threads)"`
	MiningAddrs       []string      `short:"d" long:"miningaddr" description:"Specify payment addresses for mining rewards"`
	Xpub              string        `short:"x" long:"xpub" description:"xpub address (ignored if --miningaddr is set)"`
	TestNet           bool          `long:"testnet" description:"Use testnet instead of mainnet"`
	PoolServer        string        `short:"p" long:"pool" description:"Endpoint for the pool server host:port"`
	PoolTimeout       time.Duration `short:"o" long:"timeout" default:"10s" description:"GRPC dial timeout (e.g., 5s, 1m)"`
	SlowDownDuration  time.Duration `short:"z" long:"slowDownDuration" description:"Slow down duration in seconds between each new block"`
	Generate          int           `long:"generate" description:"Number of blocks to generate (testnet only)"`
	MineOnce          bool          `long:"mineonce" description:"Mine only blocks and exit after one cycle"`
	CoinbaseScript    string        `short:"s" long:"coinbaseScript" description:"Custom Coinbase script in the format <left-bytes>:<text>:<right-bytes>"`
	BlockSiesta       time.Duration `long:"blockSiesta" description:"Pause duration between mined blocks"`
	MaxRetries        int           `long:"retryMaxAttempts" description:"Maximum number of retry attempts before giving up"`
	MaxBackoffSeconds float64       `long:"retryMaxBackoff" description:"Maximum backoff time in seconds before retrying"`
	Version           bool          `short:"v" description:"Print version"`
}

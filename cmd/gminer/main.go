// Copyright (c) 2024 The Flokicoin developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"os"

	. "github.com/flokiorg/grpc-miner/common"
	"github.com/flokiorg/grpc-miner/mining"
	"github.com/flokiorg/grpc-miner/mining/algo"
	"github.com/flokiorg/grpc-miner/mining/algo/common"
	"github.com/flokiorg/grpc-miner/mining/pb"
	"github.com/flokiorg/grpc-miner/utils"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/jessevdk/go-flags"
)

const (
	defaultPoolPort       = 80
	defaultConfigFilename = "gminer.conf"
	defaultMaxRetries     = 5
	defaultMaxBackoffSecs = 30.0
	defaultPoolTimeout    = time.Second * 30
)

var (
	parser *flags.Parser
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
}

func main() {

	var cfg Config
	parser = flags.NewParser(&cfg, flags.Default|flags.PassDoubleDash)

	if _, err := parser.Parse(); err != nil {
		os.Exit(1)
	}

	if cfg.Version {
		fmt.Println("Version:", utils.Version)
		return
	}

	configFilepath, err := utils.GetFullPath(defaultConfigFilename)
	if err != nil {
		exitWithError("unexpected error", err)
	}
	if opt := parser.FindOptionByShortName('c'); !optionDefined(opt) && utils.FileExists(configFilepath) {
		cfg.ConfigFile = configFilepath
	}

	if cfg.ConfigFile != "" {
		err := flags.NewIniParser(parser).ParseFile(cfg.ConfigFile)
		if err != nil {
			exitWithError("Failed to parse configuration file", err)
		}
	}

	logDir, err := getLogDir(cfg.ConfigFile)
	if err != nil {
		exitWithError("failed", err)
	}

	// Validate Algo
	if opt := parser.FindOptionByShortName('a'); !optionDefined(opt) {
		exitWithError("Algorithm (-a, --algo) is required but not provided.", nil)
	}
	hashAlgo, err := algo.Parse(cfg.Algo)
	if err != nil {
		exitWithError(fmt.Sprintf("invalid algo: %s", cfg.Algo), err)
	}

	// Validate mining addresses or Xpub
	if opt := parser.FindOptionByShortName('d'); !optionDefined(opt) || len(cfg.MiningAddrs) == 0 {
		if !cfg.TestNet && cfg.Xpub == "" {
			cfg.Xpub = readXpub()
		}
	} else {
		cfg.Xpub = "" // Ignore Xpub if MiningAddrs is set
	}

	// Validate Threads
	if opt := parser.FindOptionByShortName('t'); !optionDefined(opt) {
		cfg.Threads = common.DefaultThreadsMax
	}
	if cfg.Threads > common.DefaultThreadsMax {
		log.Warn().Msgf("Threads should not exceed the recommended limit: %d", common.DefaultThreadsMax)
	}

	// Validate pool endpoint
	if opt := parser.FindOptionByShortName('p'); !optionDefined(opt) {
		exitWithError("Pool endpoint (-p, --pool) is required but not provided.", nil)
	}
	if cfg.PoolServer, err = utils.ValidateAndNormalizeURI(cfg.PoolServer, defaultPoolPort); err != nil {
		exitWithError("Invalid pool endpoint", err)
	}

	// No validation needed if slowDownDuration is zero; it disables the slowdown feature.
	if cfg.SlowDownDuration < 0 {
		exitWithError(fmt.Sprintf("Invalid slowDownDuration: %v. It cannot be negative.", cfg.SlowDownDuration), nil)
	}

	var cbs *pb.CoinbaseScript
	if opt := parser.FindOptionByShortName('s'); optionDefined(opt) {
		bLeft, cbsText, bRight, err := parseCoinbaseScript(cfg.CoinbaseScript)
		if err != nil {
			exitWithError("Invalid coinbase script input", err)
		}
		cbs = &pb.CoinbaseScript{
			BytesLeft:  int64(bLeft),
			BytesRight: int64(bRight),
			Text:       cbsText,
		}
	}

	// No validation needed if slowDownDuration is zero; it disables the slowdown feature.
	if cfg.SlowDownDuration < 0 {
		exitWithError(fmt.Sprintf("Invalid slowDownDuration: %v. It cannot be negative.", cfg.SlowDownDuration), nil)
	}

	// Validate Retry Settings
	if opt := parser.FindOptionByLongName("retryMaxAttempts"); !optionDefined(opt) {
		cfg.MaxRetries = defaultMaxRetries
	}

	if opt := parser.FindOptionByLongName("retryMaxBackoff"); !optionDefined(opt) {
		cfg.MaxBackoffSeconds = defaultMaxBackoffSecs
	}

	// Validate grpc timeout
	if opt := parser.FindOptionByLongName("timeout"); !optionDefined(opt) {
		cfg.PoolTimeout = defaultPoolTimeout
	}

	fmt.Println("\nConfiguration:")
	fmt.Printf("  Algorithm: %s\n", cfg.Algo)
	fmt.Printf("  Threads: %d\n", cfg.Threads)
	if len(cfg.MiningAddrs) < 5 {
		fmt.Printf("  MiningAddrs (%d): %v\n", len(cfg.MiningAddrs), cfg.MiningAddrs)
	} else {
		fmt.Printf("  MiningAddrs (%d): %v ...\n", len(cfg.MiningAddrs), cfg.MiningAddrs[:5])
	}
	if cbs != nil {
		fmt.Printf("  CoinbaseScript: [%d:%s:%d]\n", cbs.BytesLeft, cbs.Text, cbs.BytesRight)
	}
	fmt.Printf("  TestNet: %v\n", cfg.TestNet)
	fmt.Printf("  Pool: %s\n", cfg.PoolServer)
	fmt.Print("\n\n")

	logger := utils.CreateFileLogger(filepath.Join(logDir, "gminer.log"))
	request := &pb.CandidateRequest{
		MiningAddrs:    cfg.MiningAddrs,
		Xpub:           cfg.Xpub,
		CoinbaseScript: cbs,
	}

	miner := mining.NewMiner(&cfg, hashAlgo, request, logger)

	if cfg.TestNet && cfg.Generate > 0 {
		miner.Generate(context.Background(), cfg.Generate)
	} else {
		miner.Run(context.Background())
	}

}

func readXpub() string {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter your xpub key: ")
		xpub, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input, please try again.")
			continue
		}

		// Trim whitespace and validate input
		xpub = strings.TrimSpace(xpub)
		if xpub == "" {
			fmt.Println("xpub key cannot be empty. Please enter a valid xpub.")
			continue
		}

		return xpub
	}
}

func getLogDir(configPath string) (string, error) {
	if _, err := os.Stat(configPath); err == nil {
		return filepath.Dir(configPath), nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to check config file: %w", err)
	}

	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	return filepath.Dir(exePath), nil
}

func parseCoinbaseScript(input string) (int, string, int, error) {
	parts := strings.Split(input, ":")
	if len(parts) != 3 {
		return 0, "", 0, errors.New("invalid format, expected <left-bytes>:<custom-text>:<right-bytes>")
	}

	leftBytes, err := utils.ParseIntWithDefault(parts[0], DefaultCBSBoundaryBytesSize)
	if err != nil {
		return 0, "", 0, fmt.Errorf("invalid left bytes: %v", err)
	}

	customText := parts[1]
	// if len(customText) == 0 {
	// 	return 0, "", 0, errors.New("custom-text is required and cannot be empty")
	// }

	rightBytes, err := utils.ParseIntWithDefault(parts[2], DefaultCBSBoundaryBytesSize)
	if err != nil {
		return 0, "", 0, fmt.Errorf("invalid right bytes: %v", err)
	}

	totalLength := leftBytes + len(customText) + rightBytes
	if totalLength > MaxCoinbaseScriptSize {
		return 0, "", 0, fmt.Errorf("total byte length %d exceeds maximum allowed %d", totalLength, MaxCoinbaseScriptSize)
	}

	return leftBytes, customText, rightBytes, nil
}

func exitWithError(msg string, err error) {
	log.Error().Err(err).Msg(msg)
	fmt.Println()
	parser.WriteHelp(os.Stdout)
	os.Exit(1)
}

func optionDefined(opt *flags.Option) bool {
	return opt != nil && opt.IsSet()
}

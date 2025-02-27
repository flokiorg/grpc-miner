// Copyright (c) 2024 The Flokicoin developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package network

import (
	"os"

	"github.com/flokiorg/go-flokicoin/chainutil"
	"github.com/flokiorg/go-flokicoin/rpcclient"
	"github.com/flokiorg/go-flokicoin/wire"
	"github.com/rs/zerolog/log"
)

func Listen(rpcHost, rpcUser, rpcPwd string, rpcTlsEnabled bool, rpcTlsCert string, blocks chan<- string) {

	ntfnHandlers := rpcclient.NotificationHandlers{
		OnFilteredBlockConnected: func(height int32, header *wire.BlockHeader, txns []*chainutil.Tx) {
			log.Printf("Block connected: %v (%d) %v", header.BlockHash(), height, header.Timestamp)
			blocks <- header.BlockHash().String()
		},
	}

	connCfg := &rpcclient.ConnConfig{
		Host:       rpcHost,
		Endpoint:   "ws",
		User:       rpcUser,
		Pass:       rpcPwd,
		DisableTLS: !rpcTlsEnabled,
	}

	var err error
	if rpcTlsEnabled {
		connCfg.Certificates, err = os.ReadFile(rpcTlsCert)
		if err != nil {
			log.Fatal().Err(err).Msg("unable to read cert")
		}
	}

	client, err := rpcclient.New(connCfg, &ntfnHandlers)
	if err != nil {
		log.Fatal().Err(err).Msg("notification connect")
	}

	if err := client.NotifyBlocks(); err != nil {
		log.Fatal().Err(err).Msg("failed to create rpc notification")
	}

	log.Info().Msg("notification blocks : registered")

	client.WaitForShutdown()
}

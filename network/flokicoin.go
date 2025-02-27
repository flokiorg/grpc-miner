// Copyright (c) 2024 The Flokicoin developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package network

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/flokiorg/go-flokicoin/blockchain"
	"github.com/flokiorg/go-flokicoin/chaincfg"
	"github.com/flokiorg/go-flokicoin/chaincfg/chainhash"
	"github.com/flokiorg/go-flokicoin/chainjson"
	"github.com/flokiorg/go-flokicoin/chainutil"
	"github.com/flokiorg/go-flokicoin/chainutil/hdkeychain"
	"github.com/flokiorg/go-flokicoin/rpcclient"
	"github.com/flokiorg/go-flokicoin/txscript"
	"github.com/flokiorg/go-flokicoin/wire"
	"github.com/flokiorg/grpc-miner/mining/pb"

	"github.com/rs/zerolog/log"
)

const (
	MaxCoinbaseScriptSize       = 50
	DefaultCBSBoundaryBytesSize = 5
)

type Client struct {
	*rpcclient.Client
	networkParams          *chaincfg.Params
	nextPayoutAddressIndex int
}

type BlockTemplate struct {
	Height       int64
	Amount       int64
	Transactions []*chainutil.Tx

	StrBits string
	Data    []byte

	Address chainutil.Address

	*wire.BlockHeader
}

func (mh *BlockTemplate) String() string {
	return fmt.Sprintf("height:%d txs:%d amount:%d bits:%s mr:%s", mh.Height, len(mh.Transactions), mh.Amount, mh.StrBits, mh.MerkleRoot)
}

func NewConnection(networkParams *chaincfg.Params, rpcHost, rpcUser, rpcPwd string, rpcTlsEnabled bool, rpcTlsCert string) (*Client, error) {

	c := &Client{}

	connCfg := &rpcclient.ConnConfig{
		Host: rpcHost,
		// Endpoint:     "ws",
		User:         rpcUser,
		Pass:         rpcPwd,
		HTTPPostMode: true,
		DisableTLS:   !rpcTlsEnabled,
	}

	var err error
	if rpcTlsEnabled {
		connCfg.Certificates, err = os.ReadFile(rpcTlsCert)
		if err != nil {
			log.Fatal().Err(err).Msg("unable to read cert")
		}
	}

	c.Client, err = rpcclient.New(connCfg, nil)
	if err != nil {
		return nil, err
	}

	c.networkParams = networkParams
	return c, nil
}

func (c *Client) Close() {
	c.Shutdown()
}

func (c *Client) createCoinbaseTx(address chainutil.Address, height int64, amount int64, cbs *pb.CoinbaseScript) (*chainutil.Tx, error) {

	var extraNonce []byte
	if cbs == nil {
		extraNonce = make([]byte, MaxCoinbaseScriptSize)
		if _, err := rand.Read(extraNonce); err != nil {
			return nil, err
		}
	} else {
		nonceLeft := make([]byte, cbs.BytesLeft)
		if _, err := rand.Read(nonceLeft); err != nil {
			return nil, err
		}

		nonceRight := make([]byte, cbs.BytesRight)
		if _, err := rand.Read(nonceRight); err != nil {
			return nil, err
		}

		buff := bytes.NewBuffer(nonceLeft)
		buff.Write([]byte(cbs.Text))
		buff.Write(nonceRight)

		if size := len(buff.Bytes()); size > MaxCoinbaseScriptSize {
			return nil, fmt.Errorf("coinbase script too large: %d > %d", size, MaxCoinbaseScriptSize)
		}

		extraNonce = buff.Bytes()
	}

	coinbaseScript, err := txscript.NewScriptBuilder().AddInt64(int64(height)).AddData([]byte(extraNonce)).Script()
	if err != nil {
		return nil, fmt.Errorf("failed building coinbase sig script, %w", err)
	}

	coinbaseTx := &wire.MsgTx{
		Version: 1,
		TxIn: []*wire.TxIn{{
			PreviousOutPoint: wire.OutPoint{
				Hash:  chainhash.Hash{},
				Index: 0xffffffff,
			},
			SignatureScript: coinbaseScript,
			Sequence:        0xffffffff,
		}},
		TxOut:    []*wire.TxOut{},
		LockTime: 0,
	}

	pkScript, err := txscript.PayToAddrScript(address)
	if err != nil {
		return nil, err
	}

	txOut := wire.NewTxOut(amount, pkScript)
	coinbaseTx.AddTxOut(txOut)

	return chainutil.NewTx(coinbaseTx), nil
}

func (c *Client) BuildNewHeader(nRequest *pb.CandidateRequest) (*BlockTemplate, error) {

	templateRequest := &chainjson.TemplateRequest{
		Mode: "template",
	}

	blockTemplate, err := c.GetBlockTemplate(templateRequest)
	if err != nil {
		return nil, err
	}

	address, err := getNextAddress(c.networkParams, nRequest.Xpub, nRequest.MiningAddrs, &c.nextPayoutAddressIndex, blockTemplate.Height)
	if err != nil {
		return nil, err
	}

	coinbaseTx, err := c.createCoinbaseTx(address, blockTemplate.Height, *blockTemplate.CoinbaseValue, nRequest.CoinbaseScript)
	if err != nil {
		return nil, err
	}

	var transactions []*chainutil.Tx
	transactions = append(transactions, coinbaseTx)
	for _, tx := range blockTemplate.Transactions {
		bytes, err := hex.DecodeString(tx.Data)
		if err != nil {
			return nil, err
		}

		tx, err := chainutil.NewTxFromBytes(bytes)
		if err != nil {
			return nil, err
		}

		transactions = append(transactions, tx)
	}
	merkleTree := blockchain.CalcMerkleRoot(transactions, false)

	prevBlockHash, err := chainhash.NewHashFromStr(blockTemplate.PreviousHash)
	if err != nil {
		return nil, err
	}

	bitsHex, err := strconv.ParseUint(blockTemplate.Bits, 16, 32)
	if err != nil {
		return nil, err
	}

	header := &wire.BlockHeader{
		Version:    blockTemplate.Version,
		PrevBlock:  *prevBlockHash,
		MerkleRoot: merkleTree,
		Timestamp:  time.Unix(blockTemplate.CurTime, 0),
		Bits:       uint32(bitsHex),
		Nonce:      0,
	}

	buffer := bytes.NewBuffer(nil)
	if err := header.Serialize(buffer); err != nil {
		return nil, err
	}

	return &BlockTemplate{
		Height:       blockTemplate.Height,
		Amount:       *blockTemplate.CoinbaseValue,
		StrBits:      blockTemplate.Bits,
		Data:         buffer.Bytes(),
		Transactions: transactions,
		Address:      address,
		BlockHeader:  header,
	}, nil
}

func (c *Client) GetBlock(hash string) (*chainjson.GetBlockVerboseResult, error) {

	blockHash, err := chainhash.NewHashFromStr(hash)
	if err != nil {
		return nil, err
	}

	blockhashBytes, err := json.Marshal(blockHash)
	if err != nil {
		return nil, err
	}

	rawBlock, err := c.RawRequest("getblock", []json.RawMessage{blockhashBytes})
	if err != nil {
		return nil, err
	}

	var blockResult *chainjson.GetBlockVerboseResult
	err = json.Unmarshal(rawBlock, &blockResult)
	if err != nil {
		return nil, err
	}

	return blockResult, nil

}

func (c *Client) SubmitValidBlock(vblock *pb.ValidBlock) ([]byte, error) {

	buffer := bytes.NewBuffer(vblock.Template.Block)
	block := &wire.MsgBlock{}
	if err := block.Deserialize(buffer); err != nil {
		return nil, err
	}

	block.Header.Nonce = uint32(vblock.Nonce)

	if err := c.SubmitBlock(chainutil.NewBlock(block), nil); err != nil {
		log.Error().Err(err).Msg("submiting block")
		return nil, err
	}

	headerBytes := &bytes.Buffer{}
	if err := block.Header.Serialize(headerBytes); err != nil {
		return nil, err
	}

	return headerBytes.Bytes(), nil
}

func deriveKeysFromXpub(network *chaincfg.Params, xpub string, index int64) (chainutil.Address, error) {

	extKey, err := hdkeychain.NewKeyFromString(xpub)
	if err != nil {
		return nil, fmt.Errorf("failed to parse xpub: %w", err)
	}

	childKey, err := extKey.Derive(uint32(index))
	if err != nil {
		return nil, fmt.Errorf("failed to derive child key: %w", err)
	}

	addr, err := childKey.Address(network)
	if err != nil {
		return nil, fmt.Errorf("failed to generate address: %w", err)
	}

	return addr, nil
}

func getNextAddress(network *chaincfg.Params, xpub string, addrs []string, nextAddrIndex *int, nextHeight int64) (chainutil.Address, error) {
	xpub = strings.TrimSpace(xpub)

	if len(addrs) == 0 {
		if xpub == "" {
			return nil, fmt.Errorf("payout not provided")
		}
		return deriveKeysFromXpub(network, xpub, nextHeight)
	}

	addrIndex := *nextAddrIndex
	if addrIndex < 0 || addrIndex >= len(addrs) {
		addrIndex = 0
	}

	addr, err := chainutil.DecodeAddress(addrs[addrIndex], network)
	if err != nil {
		return nil, fmt.Errorf("failed: %v got:%v", err, addrs[addrIndex])
	}

	*nextAddrIndex = (addrIndex + 1) % len(addrs)

	return addr, nil
}

package ethtxscanner

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type TxWatcher interface {
	//获取开始扫描的区块号
	GetScanStartBlock() uint64

	//获取节点地址
	GetEndpoint() string

	//是否是需要解析的tx
	IsWatchTx(from string, to string, methodId string) bool

	ProcessTx(tx *TxInfo) error
}

//tx相关信息
type TxInfo struct {
	TxHash            string
	BlockHash         string
	BlockNumber       *big.Int
	BlockUnixSecs     uint64
	From              string
	Gas               uint64
	GasPrice          *big.Int
	InputData         []byte
	Nonce             uint64
	To                string
	Value             *big.Int
	V                 []byte
	R                 []byte
	S                 []byte
	ChainID           *big.Int
	CallMethodID      string
	Status            uint64
	TransactionIndex  uint
	GasUsed           uint64
	CumulativeGasUsed uint64
}

var (
	_txWatcher             TxWatcher
	_lastScanedBlockNumber uint64 = 0
	_chainID               *big.Int
	_signer                types.EIP155Signer
)

//开始扫描
func Start(txWatcher TxWatcher) error {
	logMsg("eth tx scanner starting...")
	_txWatcher = txWatcher
	startBlock := _txWatcher.GetScanStartBlock()
	if _lastScanedBlockNumber == 0 {
		if startBlock > 0 {
			_lastScanedBlockNumber = startBlock - 1
		}
	}
	client, err := ethclient.Dial(_txWatcher.GetEndpoint())
	if err != nil {
		return err
	}
	ctx1 := context.Background()
	cid, err := client.ChainID(ctx1)
	if err != nil {
		return err
	}
	_chainID = cid
	_signer = types.NewEIP155Signer(_chainID)

	logMsg("scaning on endpoint:" + _txWatcher.GetEndpoint() + ",chainID:" + _chainID.String())

	client.Close()

	for true {
		err := scan(_lastScanedBlockNumber)
		if err != nil {
			return err
		}

		time.Sleep(1 * time.Second)
	}

	return nil
}

func scan(startBlock uint64) error {
	client, err := ethclient.Dial(_txWatcher.GetEndpoint())
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := context.Background()
	maxBn, err := client.BlockNumber(ctx)
	if err != nil {
		return err
	}
	if maxBn == _lastScanedBlockNumber {
		return nil
	}
	for _lastScanedBlockNumber < maxBn {
		logMsg("scaning block " + strconv.FormatUint(_lastScanedBlockNumber, 10) + "...")
		ctx2 := context.Background()
		currBlock := _lastScanedBlockNumber
		if _lastScanedBlockNumber > 0 {
			currBlock += 1
		}
		block, err := client.BlockByNumber(ctx2, new(big.Int).SetUint64(currBlock))
		if err != nil {
			return err
		}
		blockUnixSecs := block.Time()
		txs := block.Transactions()
		for _, tx := range txs {
			//skip contract creation tx
			if tx.To() == nil {
				continue
			}

			txData := tx.Data()
			signV, signR, signS := tx.RawSignatureValues()
			//txChainID := tx.ChainId()
			// if txChainID.Sign() != 0 {
			// 	signV = big.NewInt(int64(signV.Bytes()[0] - 35))
			// 	signV.Sub(signV, new(big.Int).Mul(txChainID, big.NewInt(2)))
			// 	signV.Add(signV, big.NewInt(27))
			// }
			// if signV.String() != "27" && signV.String() != "28" {
			// 	fmt.Println(signV.String())
			// }

			fromAddr, err := _signer.Sender(tx)
			if err != nil {
				return err
			}
			from := hexutil.Encode(fromAddr.Bytes())
			to := hexutil.Encode(tx.To().Bytes())
			methodId := ""
			if txData != nil && len(txData) >= 4 {
				methodId = hex.EncodeToString(txData[0:4])
			}
			isWatchTx := _txWatcher.IsWatchTx(from, to, methodId)
			if !isWatchTx {
				continue
			}
			txInfo := &TxInfo{
				TxHash:        tx.Hash().Hex(),
				From:          from,
				Gas:           tx.Gas(),
				GasPrice:      tx.GasPrice(),
				Nonce:         tx.Nonce(),
				To:            to,
				Value:         tx.Value(),
				V:             signV.Bytes(),
				R:             signR.Bytes(),
				S:             signS.Bytes(),
				BlockHash:     block.Hash().Hex(),
				BlockNumber:   block.Number(),
				BlockUnixSecs: blockUnixSecs,
				ChainID:       tx.ChainId(),
				CallMethodID:  methodId,
			}
			if len(txData) > 4 {
				txInfo.InputData = txData[4:]
			}

			ctx3 := context.Background()
			receipt, err := client.TransactionReceipt(ctx3, tx.Hash())
			if err != nil {
				return err
			}
			txInfo.Status = receipt.Status
			txInfo.TransactionIndex = receipt.TransactionIndex
			txInfo.GasUsed = receipt.GasUsed
			txInfo.CumulativeGasUsed = receipt.CumulativeGasUsed

			logMsg("processing tx " + txInfo.TxHash + "...")
			err = _txWatcher.ProcessTx(txInfo)
			if err != nil {
				return err
			}
		}

		_lastScanedBlockNumber = currBlock
	}

	return nil
}

func (tx *TxInfo) JSON() string {
	var sb strings.Builder
	sb.WriteString(`{"TxHash":"`)
	sb.WriteString(tx.TxHash)
	sb.WriteString(`",`)
	sb.WriteString(`"BlockHash":"`)
	sb.WriteString(tx.BlockHash)
	sb.WriteString(`",`)
	sb.WriteString(`"BlockNumber":`)
	sb.WriteString(tx.BlockNumber.String())
	sb.WriteString(`,`)
	sb.WriteString(`"BlockUnixSecs":`)
	sb.WriteString(strconv.FormatUint(tx.BlockUnixSecs, 10))
	sb.WriteString(`,`)
	sb.WriteString(`"From":"`)
	sb.WriteString(tx.From)
	sb.WriteString(`",`)
	sb.WriteString(`"Gas":`)
	sb.WriteString(strconv.FormatUint(tx.Gas, 10))
	sb.WriteString(`,`)
	sb.WriteString(`"GasPrice":`)
	sb.WriteString(tx.GasPrice.String())
	sb.WriteString(`,`)
	sb.WriteString(`"InputData":"`)
	sb.WriteString(hex.EncodeToString(tx.InputData))
	sb.WriteString(`",`)
	sb.WriteString(`"Nonce":`)
	sb.WriteString(strconv.FormatUint(tx.Nonce, 10))
	sb.WriteString(`,`)
	sb.WriteString(`"To":"`)
	sb.WriteString(tx.To)
	sb.WriteString(`",`)
	sb.WriteString(`"Value":`)
	sb.WriteString(tx.Value.String())
	sb.WriteString(`,`)
	sb.WriteString(`"V":"`)
	sb.WriteString(hex.EncodeToString(tx.V))
	sb.WriteString(`",`)
	sb.WriteString(`"R":"`)
	sb.WriteString(hex.EncodeToString(tx.R))
	sb.WriteString(`",`)
	sb.WriteString(`"S":"`)
	sb.WriteString(hex.EncodeToString(tx.S))
	sb.WriteString(`",`)
	sb.WriteString(`"ChainID":`)
	sb.WriteString(tx.ChainID.String())
	sb.WriteString(`,`)
	sb.WriteString(`"CallMethodID":"`)
	sb.WriteString(tx.CallMethodID)
	sb.WriteString(`",`)
	sb.WriteString(`"Status":`)
	sb.WriteString(strconv.FormatUint(tx.Status, 10))
	sb.WriteString(`,`)
	sb.WriteString(`"TransactionIndex":`)
	sb.WriteString(strconv.FormatUint(uint64(tx.TransactionIndex), 10))
	sb.WriteString(`,`)
	sb.WriteString(`"GasUsed":`)
	sb.WriteString(strconv.FormatUint(tx.GasUsed, 10))
	sb.WriteString(`,`)
	sb.WriteString(`"CumulativeGasUsed":`)
	sb.WriteString(strconv.FormatUint(tx.CumulativeGasUsed, 10))
	sb.WriteString(`}`)

	return sb.String()
}

func logMsg(msg string) {
	fmt.Println(time.Now().UTC().Add(time.Hour*8).Format("2006-01-02 15:04:05") + " " + msg)
}

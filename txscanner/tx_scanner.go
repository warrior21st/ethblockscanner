package txscanner

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
	GetEthClients() ([]*ethclient.Client, error)

	//是否是需要解析的tx
	IsInterestedTx(from string, to string) bool

	//tx回调处理方法
	Callback(tx *TxInfo) error

	//获取扫描间隔
	GetScanInterval() time.Duration
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

	receipt *types.Receipt
}

var (
	_txWatcher             TxWatcher
	_lastScanedBlockNumber uint64 = 0
	_chainID               *big.Int
	_signer                types.EIP155Signer
	_clientSleepTimes      map[int]int64
)

//开始扫描
func StartScanTx(txWatcher TxWatcher) error {
	LogToConsole("eth tx scanner starting...")
	_txWatcher = txWatcher
	_clientSleepTimes = make(map[int]int64)
	startBlock := _txWatcher.GetScanStartBlock()
	if _lastScanedBlockNumber == 0 {
		if startBlock > 0 {
			_lastScanedBlockNumber = startBlock - 1
		}
	}
	clients, err := _txWatcher.GetEthClients()
	if err != nil {
		return err
	}

	cid, err := clients[0].ChainID(context.Background())
	if err != nil {
		return err
	}
	_chainID = cid
	_signer = types.NewEIP155Signer(_chainID)
	LogToConsole("chainID:" + _chainID.String() + ",scaning...")

	for i := 0; i < len(clients); i++ {
		clients[i].Close()
	}

	scanInterval := _txWatcher.GetScanInterval()
	if scanInterval <= time.Millisecond {
		scanInterval = 0
	}
	errCount := 0
	for true {
		scanedBlock, err := scanTx(_lastScanedBlockNumber + 1)
		if err != nil {
			if scanedBlock > 0 {
				_lastScanedBlockNumber = scanedBlock
			} else {
				errCount++
			}
		} else {
			errCount = 0
		}

		//如果连续报错达到10次，则线程睡眠10秒后继续
		if errCount == 10 {
			LogToConsole("scaning block continuous error " + strconv.Itoa(errCount) + " times,sleep 30s...")
			time.Sleep(30 * time.Second)
			errCount = 0
		}

		if scanInterval > 0 {
			time.Sleep(scanInterval)
		}
	}

	return nil
}

func scanTx(startBlock uint64) (uint64, error) {
	clients, err := _txWatcher.GetEthClients()
	if err != nil {
		return 0, err
	}

	for i := 0; i < len(clients); i++ {
		defer clients[i].Close()
	}

	errorSleepSeconds := int64(10)
	currBlock := startBlock
	finishedBlock := startBlock - 1
	for true {
		avaiIndexes := RebuildAvaiIndexes(len(clients), &_clientSleepTimes)
		if len(avaiIndexes) == 0 {
			break
		}

		index := avaiIndexes[currBlock%uint64(len(avaiIndexes))]
		client := clients[index]
		LogToConsole("scaning block " + strconv.FormatUint(currBlock, 10) + " txs on client_" + strconv.Itoa(index) + "...")

		block, err := client.BlockByNumber(context.Background(), new(big.Int).SetUint64(currBlock))
		if err != nil && err.Error() != "not found" {
			_clientSleepTimes[index] = time.Now().UTC().Unix() + errorSleepSeconds
			avaiIndexes = RebuildAvaiIndexes(len(clients), &_clientSleepTimes)

			LogToConsole("client_" + strconv.Itoa(index) + "response error,sleep " + strconv.FormatInt(errorSleepSeconds, 10) + "s.")
			continue
		}

		if block == nil {
			break
		}

		blockUnixSecs := block.Time()
		txs := block.Transactions()
		resolveTxError := false
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
				return 0, err
			}
			from := strings.ToLower(hexutil.Encode(fromAddr.Bytes()))
			to := strings.ToLower(hexutil.Encode(tx.To().Bytes()))
			methodId := ""
			if txData != nil && len(txData) >= 4 {
				methodId = hex.EncodeToString(txData[0:4])
			}
			if !_txWatcher.IsInterestedTx(from, to) {
				continue
			}
			txInfo := &TxInfo{
				TxHash:        strings.ToLower(tx.Hash().Hex()),
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

			receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
			if err != nil {
				_clientSleepTimes[index] = time.Now().UTC().Unix() + errorSleepSeconds
				avaiIndexes = RebuildAvaiIndexes(len(clients), &_clientSleepTimes)

				LogToConsole("client_" + strconv.Itoa(index) + "reponse error,sleep " + strconv.FormatInt(errorSleepSeconds, 10) + "s.")
				resolveTxError = true
				break
			}

			txInfo.receipt = receipt
			txInfo.Status = receipt.Status
			txInfo.TransactionIndex = receipt.TransactionIndex
			txInfo.GasUsed = receipt.GasUsed
			txInfo.CumulativeGasUsed = receipt.CumulativeGasUsed

			LogToConsole("processing tx " + txInfo.TxHash + "...")
			err = _txWatcher.Callback(txInfo)
			if err != nil {
				return finishedBlock, err
			}
		}

		if !resolveTxError {
			finishedBlock = currBlock
			currBlock++
		}
	}

	return finishedBlock, nil
}

//获取tx logs
func (tx *TxInfo) Logs() []*types.Log {
	return tx.receipt.Logs
}

//获取tx的json形式
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

func LogToConsole(msg string) {
	fmt.Println(time.Now().Add(8*time.Hour).Format("2006-01-02 15:04:05") + "  " + msg)
}

func RebuildAvaiIndexes(clientsCount int, clientSleepTimes *map[int]int64) []int {
	avaiIndexes := make([]int, 0, clientsCount)
	for i := 0; i < clientsCount; i++ {
		if time.Now().UTC().Unix() < (*clientSleepTimes)[i] {
			continue
		}
		avaiIndexes = append(avaiIndexes, i)
	}

	return avaiIndexes
}

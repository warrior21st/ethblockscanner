package txlogscanner

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	_txlogWatcher          TxlogWatcher
	_lastScanedBlockNumber uint64 = 0
	_lastBlockForwardTime int64 = 0
	_chainID               *big.Int
	_signer                types.EIP155Signer
	_clientSleepTimes      map[int]int64
)

type TxlogWatcher interface {
	//获取开始扫描的区块号
	GetScanStartBlock() uint64

	//获取节点地址
	GetEthClients() ([]*ethclient.Client, error)

	//是否是需要解析的tx
	IsInterestedLog(address string, topic0 string) bool

	//tx log回调处理方法
	Callback(txlog *types.Log) error

	//获取扫描间隔
	GetScanInterval() time.Duration
}

//开始扫描
func StartScanTxLogs(txlogWatcher TxlogWatcher) error {
	LogToConsole("eth tx log scanner starting...")
	_txlogWatcher = txlogWatcher
	_clientSleepTimes = make(map[int]int64)
	startBlock := _txlogWatcher.GetScanStartBlock()
	if _lastScanedBlockNumber == 0 {
		if startBlock > 0 {
			_lastScanedBlockNumber = startBlock - 1
		}
	}
	clients, err := _txlogWatcher.GetEthClients()
	if err != nil {
		return err
	}

	cid, err := clients[0].ChainID(context.Background())
	if err != nil {
		return err
	}
	_chainID = cid
	_signer = types.NewEIP155Signer(_chainID)
	LogToConsole("chainID:" + _chainID.String() + ",filter scaning...")

	for i := 0; i < len(clients); i++ {
		clients[i].Close()
	}

	scanInterval := _txlogWatcher.GetScanInterval()
	if scanInterval <= time.Millisecond {
		scanInterval = 0
	}
	errCount := 0
	for true {		
		scanedBlock, err := scanTxLogs(_lastScanedBlockNumber + 1)
		if err != nil {
			if scanedBlock > 0 {
				_lastScanedBlockNumber = scanedBlock
			} else {
				errCount++
			}
		} else {
			_lastScanedBlockNumber = scanedBlock
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

func scanTxLogs(startBlock uint64) (uint64, error) {
	clients, err := _txlogWatcher.GetEthClients()
	if err != nil {
		return 0, err
	}

	for i := 0; i < len(clients); i++ {
		defer clients[i].Close()
	}

	errorSleepSeconds := int64(10)
	currBlock := startBlock
	finisedMaxBlock := startBlock - 1
	filter := ethereum.FilterQuery{}
	for true {
		avaiIndexes := RebuildAvaiIndexes(len(clients), &_clientSleepTimes)
		if len(avaiIndexes) == 0 {
			break
		}
		index := avaiIndexes[currBlock%uint64(len(avaiIndexes))]
		client := clients[index]
		LogToConsole("scaning block " + strconv.FormatUint(currBlock, 10) + " tx logs on client_" + strconv.Itoa(index) + "...")

		filter.FromBlock = new(big.Int).SetUint64(currBlock)
		filter.ToBlock = new(big.Int).SetUint64(currBlock)
		//filter.BlockHash = &currBlockHash
		logs, err := client.FilterLogs(context.Background(), filter)
		if err != nil {
			_clientSleepTimes[index] = time.Now().UTC().Unix() + errorSleepSeconds
			LogToConsole("client_" + strconv.Itoa(index) + "response error,sleep " + strconv.FormatInt(errorSleepSeconds, 10) + "s.")
			continue
		}

		if logs == nil || len(logs) == 0 {
			blockNotMined:=true
			if time.Now().Unix() - _lastBlockForwardTime>=20{
				blockNumber,err:=client.BlockNumber(context.Background())
				if err!=nil{
					_clientSleepTimes[index] = time.Now().UTC().Unix() + errorSleepSeconds
					LogToConsole("client_" + strconv.Itoa(index) + "response error,sleep " + strconv.FormatInt(errorSleepSeconds, 10) + "s.")
					continue
				}

				blockNotMined=blockNumber < currBlock
			}else{

			}
			if(blockNotMined){
				LogToConsole("block " + strconv.FormatUint(currBlock, 10) + " is not mined or not synced on client_" + strconv.Itoa(index) + ".")
				break
			}else{
				currBlock++
				continue
			}
		}

		for _, log := range logs {
			if _txlogWatcher.IsInterestedLog(log.Address.Hex(), log.Topics[0].Hex()) {
				err = _txlogWatcher.Callback(&log)
				if err != nil {
					return finisedMaxBlock, err
				}
			}
		}

		_lastBlockForwardTime = time.Now().Unix()
		finisedMaxBlock = currBlock
		currBlock++
	}

	return finisedMaxBlock, nil
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

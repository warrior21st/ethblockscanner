package txlogscanner

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	_txlogWatcher          TxlogWatcher
	_lastScanedBlockNumber uint64 = 0
	_lastBlockForwardTime  int64  = 0
	_clientSleepTimes      map[int]int64
)

type TxlogWatcher interface {
	//获取开始扫描的区块号
	GetScanStartBlock() uint64

	//获取节点地址
	GetEthClients() ([]*ethclient.Client, error)

	//获取单次扫描区块数
	GetPerScanBlockCount() uint64

	GetInterestedAddresses() []common.Address

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

	for i := 0; i < len(clients); i++ {
		defer clients[i].Close()
	}

	scanInterval := _txlogWatcher.GetScanInterval()
	if scanInterval <= time.Millisecond {
		scanInterval = 0
	}
	errCount := 0
	for true {
		scanedBlock, err := scanTxLogs(clients[0], _lastScanedBlockNumber+1)
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

func scanTxLogs(client *ethclient.Client, startBlock uint64) (uint64, error) {

	perScanIncrment := _txlogWatcher.GetPerScanBlockCount() - 1
	currBlock := startBlock
	filter := ethereum.FilterQuery{
		Addresses: _txlogWatcher.GetInterestedAddresses(),
	}

	blockHeight := getBlockNumber(client)
	LogToConsole(fmt.Sprintf("current block height: %d", blockHeight))

	filter.FromBlock = new(big.Int).SetUint64(currBlock)
	filter.ToBlock = new(big.Int).SetUint64(currBlock + perScanIncrment)
	if uint64(filter.ToBlock.Int64()) > blockHeight {
		filter.ToBlock = big.NewInt(int64(blockHeight))
	}

	LogToConsole(fmt.Sprintf("scaning block %s - %s tx logs...", filter.FromBlock.String(), filter.ToBlock.String()))

	logs, err := client.FilterLogs(context.Background(), filter)
	for err != nil {
		LogToConsole(fmt.Sprintf("get logs error: %s,sleep 1s...", err.Error()))
		time.Sleep(time.Second)
		logs, err = client.FilterLogs(context.Background(), filter)
	}

	for _, log := range logs {
		if _txlogWatcher.IsInterestedLog(log.Address.Hex(), log.Topics[0].Hex()) {
			err = _txlogWatcher.Callback(&log)
			if err != nil {
				panic(err)
			}
		}
	}

	return filter.ToBlock.Uint64() + 1, nil
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

func getBlockNumber(client *ethclient.Client) uint64 {
	blockNumber, err := client.BlockNumber(context.Background())
	for err != nil {
		LogToConsole(fmt.Sprintf("get block height error: %s,sleep 1s...", err.Error()))
		time.Sleep(time.Second)
		blockNumber, err = client.BlockNumber(context.Background())
	}

	return blockNumber
}

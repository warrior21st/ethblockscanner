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
	// _txlogWatcher          TxlogWatcher
	// _lastScanedBlockNumber uint64 = 0
	_lastBlockForwardTime int64 = 0
	// _clientSleepTimes      map[int]int64
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
	// _clientSleepTimes = make(map[int]int64)
	startBlock := txlogWatcher.GetScanStartBlock()
	if startBlock > 0 {
		startBlock = startBlock - 2
	}
	lastScanedBlockNumber := uint64(0)
	if startBlock > 0 {
		lastScanedBlockNumber = startBlock
	}
	clients, err := txlogWatcher.GetEthClients()
	if err != nil {
		return err
	}

	for i := 0; i < len(clients); i++ {
		defer clients[i].Close()
	}

	scanInterval := txlogWatcher.GetScanInterval()
	if scanInterval <= time.Millisecond {
		scanInterval = 0
	}
	errCount := 0
	for true {
		scanedBlock, err := scanTxLogs(clients[0], lastScanedBlockNumber+1, txlogWatcher)
		if err != nil {
			if scanedBlock > 0 {
				lastScanedBlockNumber = scanedBlock
			} else {
				errCount++
			}
		} else {
			lastScanedBlockNumber = scanedBlock
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

func scanTxLogs(client *ethclient.Client, startBlock uint64, txlogWatcher TxlogWatcher) (uint64, error) {

	// currBlock := startBlock
	filter := ethereum.FilterQuery{
		Addresses: txlogWatcher.GetInterestedAddresses(),
	}

	blockHeight := getBlockNumber(client)
	LogToConsole(fmt.Sprintf("current block height: %d", blockHeight))

	if startBlock > blockHeight {
		LogToConsole(fmt.Sprintf("block %d not minted,sleep 1s...", startBlock))
		time.Sleep(time.Second)
		return startBlock - 1, nil
	}

	filter.FromBlock = new(big.Int).SetUint64(startBlock)
	filter.ToBlock = new(big.Int).SetUint64(startBlock + txlogWatcher.GetPerScanBlockCount())
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
		if txlogWatcher.IsInterestedLog(log.Address.Hex(), log.Topics[0].Hex()) {
			err = txlogWatcher.Callback(&log)
			if err != nil {
				panic(err)
			}
		}
	}

	return filter.ToBlock.Uint64(), nil
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

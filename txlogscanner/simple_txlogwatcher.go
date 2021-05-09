package txlogscanner

import (
	"encoding/base64"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

//简单交易管理结构
type SimpleTxLogWatcher struct {
	endpoints      []string
	infuraSecrets  []string
	scanStartBlock uint64
	interestedLogs map[string]interface{}
	scanInterval   time.Duration
	callback       func(*types.Log) error
}

//构造一个新的简单tx管理结构(默认3秒钟扫描一次)
func NewSimpleTxLogWatcher(endpoints []string, scanStartBlock uint64, scanInterval time.Duration, callback func(*types.Log) error) *SimpleTxLogWatcher {

	return &SimpleTxLogWatcher{
		endpoints:      endpoints,
		scanStartBlock: scanStartBlock,
		scanInterval:   scanInterval,
		callback:       callback,
	}
}

func (watcher *SimpleTxLogWatcher) SetInfuraSecrets(secrets []string) {
	watcher.infuraSecrets = secrets
}

//添加关注的from address
func (watcher *SimpleTxLogWatcher) AddInterestedLog(address string, topic0 string) {
	if watcher.interestedLogs == nil {
		watcher.interestedLogs = make(map[string]interface{})
	}
	watcher.interestedLogs[strings.ToLower(address+"_"+topic0)] = true
}

func (watcher *SimpleTxLogWatcher) GetScanStartBlock() uint64 {

	return watcher.scanStartBlock
}

func (watcher *SimpleTxLogWatcher) GetEthClients() ([]*ethclient.Client, error) {
	clients := make([]*ethclient.Client, len(watcher.endpoints))
	for i := 0; i < len(watcher.endpoints); i++ {
		rpcClient, err := rpc.Dial(watcher.endpoints[i])
		if err != nil {
			return nil, err
		}
		if i < len(watcher.infuraSecrets) && strings.Trim(watcher.infuraSecrets[i], " ") != "" {
			rpcClient.SetHeader("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(":"+watcher.infuraSecrets[i])))
		}

		clients[i] = ethclient.NewClient(rpcClient)
	}

	return clients, nil
}

func (watcher *SimpleTxLogWatcher) IsInterestedTx(addr string, topic0 string) bool {

	return watcher.interestedLogs[strings.ToLower(addr+"_"+topic0)] != nil
}

//tx回调处理方法
func (watcher *SimpleTxLogWatcher) Callback(tx *types.Log) error {
	return watcher.callback(tx)
}

//获取区块扫描间隔
func (watcher *SimpleTxLogWatcher) GetScanInterval() time.Duration {
	return watcher.scanInterval
}

//设置区块扫描间隔
func (watcher *SimpleTxLogWatcher) SetScanInterval(interval time.Duration) {
	watcher.scanInterval = interval
}

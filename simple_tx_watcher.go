package ethtxscanner

import (
	"encoding/base64"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

//简单交易管理结构
type SimpleTxWatcher struct {
	endpoints       []string
	infuraSecrets   []string
	scanStartBlock  uint64
	interestedFroms map[string]interface{}
	interestedTos   map[string]interface{}
	scanInterval    time.Duration
	callback        func(*TxInfo) error
}

//构造一个新的简单tx管理结构(默认3秒钟扫描一次)
func NewSimpleTxWatcher(endpoints []string, scanStartBlock uint64, scanInterval time.Duration, callback func(*TxInfo) error) *SimpleTxWatcher {

	return &SimpleTxWatcher{
		endpoints:      endpoints,
		scanStartBlock: scanStartBlock,
		scanInterval:   scanInterval,
		callback:       callback,
	}
}

func (watcher *SimpleTxWatcher) SetInfuraSecrets(secrets []string) {
	watcher.infuraSecrets = secrets
}

//添加关注的from address
func (watcher *SimpleTxWatcher) AddInterestedFrom(from string) {
	if watcher.interestedFroms == nil {
		watcher.interestedFroms = make(map[string]interface{})
	}
	watcher.interestedFroms[strings.ToLower(from)] = true
}

//添加关注的to address
func (watcher *SimpleTxWatcher) AddInterestedTo(to string) {
	if watcher.interestedTos == nil {
		watcher.interestedTos = make(map[string]interface{})
	}
	watcher.interestedTos[strings.ToLower(to)] = true
}

func (watcher *SimpleTxWatcher) GetScanStartBlock() uint64 {

	return watcher.scanStartBlock
}

func (watcher *SimpleTxWatcher) GetEthClients() ([]*ethclient.Client, error) {
	var clients []*ethclient.Client
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

func (watcher *SimpleTxWatcher) IsInterestedTx(from string, to string) bool {

	if watcher.interestedFroms != nil {
		_, b := watcher.interestedFroms[strings.ToLower(from)]
		if b {
			return b
		}
	}
	if watcher.interestedTos != nil {
		_, b := watcher.interestedTos[strings.ToLower(to)]
		if b {
			return b
		}
	}

	return false
}

//tx回调处理方法
func (watcher *SimpleTxWatcher) Callback(tx *TxInfo) error {
	return watcher.callback(tx)
}

//获取区块扫描间隔
func (watcher *SimpleTxWatcher) GetScanInterval() time.Duration {
	return watcher.scanInterval
}

//设置区块扫描间隔
func (watcher *SimpleTxWatcher) SetScanInterval(interval time.Duration) {
	watcher.scanInterval = interval
}

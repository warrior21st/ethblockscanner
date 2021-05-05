package ethtxscanner

import (
	"encoding/base64"
	"strings"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

//简单交易管理结构
type SimpleTxWatcher struct {
	endpoint        string
	infuraSecret    string
	scanStartBlock  uint64
	interestedFroms map[string]interface{}
	interestedTos   map[string]interface{}
	callback        func(*TxInfo) error
}

//构造一个新的简单tx管理结构
func NewSimpleTxWatcher(endpoint string, scanStartBlock uint64, callback func(*TxInfo) error) *SimpleTxWatcher {

	return &SimpleTxWatcher{
		endpoint:       endpoint,
		scanStartBlock: scanStartBlock,
		callback:       callback,
	}
}

func (watcher *SimpleTxWatcher) SetInfuraSecret(secret string) {
	watcher.infuraSecret = secret
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

func (watcher *SimpleTxWatcher) GetEthClient() (*ethclient.Client, error) {
	rpcClient, err := rpc.Dial(watcher.endpoint)
	if err != nil {
		return nil, err
	}

	if strings.Trim(watcher.endpoint, " ") != "" {
		rpcClient.SetHeader("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(":"+watcher.infuraSecret)))
	}

	return ethclient.NewClient(rpcClient), nil
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

func (watcher *SimpleTxWatcher) Callback(tx *TxInfo) error {

	return watcher.callback(tx)
}

package ethtxscanner

import (
	"errors"
	"strings"
)

//简单交易管理结构
type SimpleTxWatcher struct {
	endpoint          string
	scanStartBlock    uint64
	watchFroms        map[string]func(*TxInfo) error
	watchToAndMethods map[string]map[string]func(*TxInfo) error
}

//构造一个新的简单tx管理结构
func NewSimpleTxWatcher(endpoint string, scanStartBlock uint64) *SimpleTxWatcher {

	return &SimpleTxWatcher{
		endpoint:       endpoint,
		scanStartBlock: scanStartBlock,
	}
}

//观察某个from address的交易
func (watcher *SimpleTxWatcher) WatchFrom(from string, handler func(*TxInfo) error) {
	if watcher.watchFroms == nil {
		watcher.watchFroms = make(map[string]func(*TxInfo) error)
	}
	watcher.watchFroms[strings.ToLower(from)] = handler
}

//观察某个to address的所有交易
func (watcher *SimpleTxWatcher) WatchTo(to string, handler func(*TxInfo) error) {
	toLow := strings.ToLower(to)
	if watcher.watchToAndMethods == nil {
		watcher.watchToAndMethods = make(map[string]map[string]func(*TxInfo) error)
	}
	_, b := watcher.watchToAndMethods[toLow]
	if !b {
		watcher.watchToAndMethods[toLow] = make(map[string]func(*TxInfo) error)
	}
	watcher.watchToAndMethods[toLow]["all"] = handler
}

//观察某个to address的特定方法id的交易
func (watcher *SimpleTxWatcher) WatchToAndMethods(to string, methodIds []string, handler func(*TxInfo) error) {
	toLow := strings.ToLower(to)
	if watcher.watchToAndMethods == nil {
		watcher.watchToAndMethods = make(map[string]map[string]func(*TxInfo) error)
	}
	for _, methodId := range methodIds {
		_, b := watcher.watchToAndMethods[toLow]
		if !b {
			watcher.watchToAndMethods[toLow] = make(map[string]func(*TxInfo) error)
		}

		watcher.watchToAndMethods[toLow][strings.ToLower(methodId)] = handler
	}
}

func (watcher *SimpleTxWatcher) GetScanStartBlock() uint64 {

	return watcher.scanStartBlock
}

func (watcher *SimpleTxWatcher) GetEndpoint() string {

	return watcher.endpoint
}

func (watcher *SimpleTxWatcher) IsWatchTx(from string, to string, methodId string) bool {

	if watcher.watchFroms != nil {
		_, b := watcher.watchFroms[from]
		if b {
			return b
		}
	}
	if watcher.WatchToAndMethods != nil {
		_, b := watcher.watchToAndMethods[to]
		if b {
			_, b = watcher.watchToAndMethods[to][methodId]
			if b {
				return b
			}
			_, b = watcher.watchToAndMethods[to]["all"]
			if b {
				return b
			}
		}
	}

	return false
}

func (watcher *SimpleTxWatcher) ProcessTx(tx *TxInfo) error {

	if _, exist := watcher.watchToAndMethods[tx.To]; exist {
		if _, exist := watcher.watchToAndMethods[tx.To][tx.CallMethodID]; exist {
			return watcher.watchToAndMethods[tx.To][tx.CallMethodID](tx)
		} else if _, exist := watcher.watchToAndMethods[tx.To]["all"]; exist {
			return watcher.watchToAndMethods[tx.To]["all"](tx)
		} else if _, exist := watcher.watchFroms[tx.From]; exist {
			return watcher.watchFroms[tx.From](tx)
		}
	}

	if _, exist := watcher.watchFroms[tx.From]; exist {
		return watcher.watchFroms[tx.From](tx)
	} else {
		return errors.New("can not find tx handler,from:" + tx.From + ",to:" + tx.To + ",methodId:" + tx.CallMethodID)
	}
}

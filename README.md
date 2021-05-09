# ethereum block scanner
### pkg
    import (
		"github.com/warrior21st/ethblockscanner/txlogscanner"
		"github.com/warrior21st/ethblockscanner/txscanner"
	)
### txscanner sample
	usdtAddr := "0xdac17f958d2ee523a2206206994597c13d831ec7"
	endpoints:=[]string{ "https://mainnet.infura.io/v3/[your infura project 1 ID]", "https://mainnet.infura.io/v3/[your infura project 2 ID]",...}
	secrets:=[]string{ "[your infura project 1 secret]", "[your infura project 2 secret]",...}
	interval := 1 * time.Second
	
	//txscanner
	txWatcher := txscanner.NewSimpleTxWatcher(endpoints, 12400770, interval, func(tx *txscanner.TxInfo) error {
		transferMethodID := hex.EncodeToString(crypto.Keccak256([]byte("transfer(address,uint256)"))[:4])
		if tx.CallMethodID != transferMethodID {
			return nil
		}

		abiJsonStr := jsonutil.ReadJsonValue(commonutil.ReadFileBytes(commonutil.MapPath("/contractabis/ERC20.json")), "abi")
		contractAbi, err := abi.JSON(strings.NewReader(abiJsonStr))
		if err != nil {
			panic(err)
		}
		if tx.Status == 0 {
			return nil
		}

		var transferEvent struct {
			From  common.Address
			To    common.Address
			Value *big.Int
		}

		err = contractAbi.UnpackIntoInterface(&transferEvent, "Transfer", tx.Logs()[0].Data)
		if err != nil {
			fmt.Println("Failed to unpack")
			return err
		}
		transferEvent.From = common.BytesToAddress(tx.Logs()[0].Topics[1].Bytes())
		transferEvent.To = common.BytesToAddress(tx.Logs()[0].Topics[2].Bytes())

		fmt.Println("Transfer  from:" + transferEvent.From.Hex() + ",to:" + transferEvent.To.Hex() + ",value:" + transferEvent.Value.String())

		return nil
	})

	txWatcher.SetInfuraSecrets(secrets)
	txWatcher.AddInterestedTo(usdtAddr)
	txscanner.StartScanTx(txWatcher)
	
	//txlogscanner
	watcher := txlogscanner.NewSimpleTxLogWatcher(endpoints, 12400629, interval, func(log *types.Log) error {

		var transferEvent struct {
			From  common.Address
			To    common.Address
			Value *big.Int
		}

		abiJsonStr := jsonutil.ReadJsonValue(commonutil.ReadFileBytes(commonutil.MapPath("/contractabis/ERC20.json")), "abi")
		contractAbi, err := abi.JSON(strings.NewReader(abiJsonStr))
		err = contractAbi.UnpackIntoInterface(&transferEvent, "Transfer", log.Data)
		if err != nil {
			fmt.Println("Failed to unpack")
			return err
		}
		transferEvent.From = common.BytesToAddress(log.Topics[1].Bytes())
		transferEvent.To = common.BytesToAddress(log.Topics[2].Bytes())

		fmt.Println("txhash:" + log.TxHash.Hex() + " Transfer  from:" + transferEvent.From.Hex() + ",to:" + transferEvent.To.Hex() + ",value:" + transferEvent.Value.String())

		return nil
	})

	watcher.SetInfuraSecrets(secrets)
	watcher.AddInterestedParams(usdtAddr, "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	txlogscanner.StartScanTxLogs(watcher)


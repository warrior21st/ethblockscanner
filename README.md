# ethereum block transactions scanner
### pkg
    import "github.com/warrior21st/ethtxscanner"
### Sample code
	infuraEndpoint:=[your infura endpoint]
	infuraSecret:=[your infura endpoint]
	txWatcher := ethtxscanner.NewSimpleTxWatcher(infuraEndpoint, 11358668, func(tx *ethtxscanner.TxInfo) error {
		transferMethodID := hex.EncodeToString(crypto.Keccak256([]byte("transfer(address,uint256)"))[:4])
		if tx.CallMethodID != transferMethodID {
			return nil
		}

		jsonStr := tx.JSON()
		fmt.Println("txinfos:" + jsonStr)
		fmt.Println(tx.Logs())
		abiJsonStr := jsonutil.ReadJsonValue(commonutil.ReadFileBytes(commonutil.MapPath("/contractabis/ERC20.json")), "abi")
		contractAbi, err := abi.JSON(strings.NewReader(abiJsonStr))
		if err != nil {
			panic(err)
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

	txWatcher.SetInfuraSecret(infuraSecret)
	txWatcher.AddInterestedTo(usdtAddr)

	ethtxscanner.Start(txWatcher)


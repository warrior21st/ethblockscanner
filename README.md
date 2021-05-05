# ethereum block transactions scanner
### pkg
    import "github.com/warrior21st/ethtxscanner"
### Sample code
	usdtAddr := "0xdac17f958d2ee523a2206206994597c13d831ec7"
	endpoints:=[2]string{ "https://mainnet.infura.io/v3/[your infura project 1 ID]", "https://mainnet.infura.io/v3/[your infura project 2 ID]"}
	secrets:=[2]string{ "[your infura project 1 secret]", "[your infura project 2 secret]"}
	interval := 1 * time.Second
	txWatcher := ethtxscanner.NewSimpleTxWatcher(endpoints, 11358668, interval, func(tx *ethtxscanner.TxInfo) error {
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

	txWatcher.SetInfuraSecrets(secrets)
	txWatcher.AddInterestedTo(usdtAddr)

	ethtxscanner.Start(txWatcher)


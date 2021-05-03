# ethereum block transactions scanner
### pkg
    import "github.com/warrior21st/ethtxscanner"
### Sample code
	endpoint := "https://mainnet.infura.io"
	usdtAddr := "0xdac17f958d2ee523a2206206994597c13d831ec7"

	txWatcher := ethtxscanner.NewSimpleTxWatcher(endpoint, 11358668,func(tx *ethtxscanner.TxInfo) error {
		transferMethodID:=hex.EncodeToString(crypto.Keccak256([]byte("transfer(address,uint256)"))[:4])
		if (tx.CallMethodID!=transferMethodID){
			return nil
		}

		jsonStr := tx.JSON()
		fmt.Println("txinfos:" + jsonStr)
		fmt.Println(tx.Logs())
		abiJsonStr := commutil.ReadJsonValue(commutil.ReadFileBytes(commutil.MapPath("/contractabis/ERC20.json")), "abi")
		contractAbi, err := abi.JSON(strings.NewReader(abiJsonStr))
		if err != nil {
			panic(err)
		}
		log, err := contractAbi.Unpack("Transfer", tx.Logs()[0].Data)
		if err != nil {
			panic(err)
		}
		fmt.Println(log)

		utcNow := time.Now().UTC()
		path := commutil.MapPath("/txlogs")

		if !commutil.IsExistPath(path) {
			os.Mkdir(path, 0755)
		}
		path = commutil.CombinePath(path, commutil.TimeToDateString(&utcNow)+".log")
		file, _ := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
		defer file.Close()

		buffer := bufio.NewWriter(file)
		buffer.WriteString(time.Now().UTC().Add(time.Hour*8).Format("2006-01-02 15:04:05") + " " + jsonStr + "\n")
		buffer.Flush()

		return nil
	})
	txWatcher.AddInterestedTo(usdtAddr)

	ethtxscanner.Start(txWatcher)


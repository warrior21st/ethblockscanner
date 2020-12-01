# ethereum block transactions scanner
### pkg
    import "github.com/warrior21st/ethtxscaner"
### Sample code
    endpoint := "https://mainnet.infura.io"
    usdtAddr := "0xdac17f958d2ee523a2206206994597c13d831ec7"
    methodIds := []string{hex.EncodeToString(crypto.Keccak256([]byte("transfer(address,uint256)"))[:4])}
    txWatcher := ethtxlistener.NewSimpleTxWatcher(endpoint, 11358668)
    txWatcher.WatchToAndMethods(usdtAddr, methodIds, func(tx *TxInfo) error {
        jsonStr := tx.JSON()
        fmt.Println("txinfos:" + jsonStr)

		// output：
		// {
		// 	"TxHash":"0x9dbb3cdfd37e31a68fceebdbe7696731557f17197f6753b77d3d47579fd0c7a4",
		// 	"BlockHash":"0x9bf9369c52aae2f40186d181720424ef0edae2ebf97d602f472d0309259dad06",
		// 	"BlockNumber":11358669,
		// 	"BlockUnixSecs":1606720456,
		// 	"From":"0xadb2b42f6bd96f5c65920b9ac88619dce4166f94",
		// 	"Gas":100000,
		// 	"GasPrice":28000000000,
		// 	"InputData":"0000000000000000000000006a689b567e350a8b5aafdd46deb6ff71106a404400000000000000000000000000000000000000000000000000000000c499e39a",
		// 	"Nonce":1368848,
		// 	"To":"0xdac17f958d2ee523a2206206994597c13d831ec7",
		// 	"Value":0,
		// 	"V":"25",
		// 	"R":"09c0027a8f60cb78f463ddea32a99e8e72aa66ecc07012583267e90a39357707",
		// 	"S":"25952548d37fd9924556ae5fa37c2488d127384a143adbfa12771dfc28a25eab",
		// 	"ChainID":1,
		// 	"CallMethodID":"a9059cbb",
		// 	"Status":1,
		// 	"TransactionIndex":148,
		// 	"GasUsed":56209,
		// 	"CumulativeGasUsed":11655570
		// }

        return nil
    })

    ethtxlistener.Start(txWatcher)


package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/tdex-network/tdex-daemon/pkg/explorer/esplora"
	"github.com/tdex-network/tdex-daemon/pkg/trade"
	tradeclient "github.com/tdex-network/tdex-daemon/pkg/trade/client"
	trademarket "github.com/tdex-network/tdex-daemon/pkg/trade/market"
	"github.com/vulpemventures/go-elements/network"
)

func main() {
	explorerSvc, err := esplora.NewService("http://localhost:3001", 15000)
	if err != nil {
		log.Fatal(err)
	}

	tradeClient, err := tradeclient.NewTradeClient("localhost", 9945)
	if err != nil {
		log.Fatal(err)
	}

	wallet, err := trade.NewRandomWallet(&network.Regtest)
	if err != nil {
		log.Fatal(err)
	}
	privkey := wallet.PrivateKey()
	addr := wallet.Address()
	blindkey := wallet.BlindingKey()

	fmt.Println("Wallet info")
	fmt.Println("private key:", hex.EncodeToString(privkey))
	fmt.Println("blinding key:", hex.EncodeToString(blindkey))
	fmt.Println("address:", addr)

	utxos, err := explorerSvc.GetUnspentsForAddresses([]string{addr}, [][]byte{blindkey})
	if err != nil {
		log.Fatal(err)
	}
	if len(utxos) == 0 {
		fmt.Println("sending some funds to wallet...")
		if _, err := explorerSvc.Faucet(addr, 1, network.Regtest.AssetID); err != nil {
			log.Fatal(err)
		}

		for {
			utxos, err := explorerSvc.GetUnspentsForAddresses([]string{addr}, [][]byte{blindkey})
			if err != nil {
				log.Fatal(err)
			}
			if len(utxos) > 0 {
				break
			}
			time.Sleep(time.Second)
		}
	}
	tt, err := trade.NewTrade(trade.NewTradeOpts{
		ExplorerService: explorerSvc,
		Chain:           "regtest",
		Client:          tradeClient,
	})
	if err != nil {
		log.Fatal(err)
	}

	res, err := tradeClient.Markets()
	if err != nil {
		log.Fatal(err)
	}
	if len(res.GetMarkets()) <= 0 {
		log.Fatal("no markets found")
	}

	market := trademarket.Market{
		BaseAsset:  res.GetMarkets()[0].GetMarket().GetBaseAsset(),
		QuoteAsset: res.GetMarkets()[0].GetMarket().GetQuoteAsset(),
	}

	// sell 0.05 LBTC
	amountInSatoshi := 5000000
	amountInBtc := float64(amountInSatoshi) / 100000000

	start := time.Now()
	txid, err := tt.SellAndComplete(trade.BuyOrSellAndCompleteOpts{
		Market:      market,
		Amount:      uint64(amountInSatoshi),
		Asset:       market.BaseAsset,
		PrivateKey:  privkey,
		BlindingKey: blindkey,
	})
	if err != nil {
		log.Fatal(err)
	}
	elapsed := time.Since(start)
	fmt.Printf("TRADE TOOK %fs\n", elapsed.Seconds())
	fmt.Printf("sold %.2f LBTC in tx %s\n", amountInBtc, txid)

	time.Sleep(5 * time.Second)

	amountInSatoshi -= 2000000
	amountInBtc = float64(amountInSatoshi) / 100000000

	start = time.Now()
	txid, err = tt.BuyAndComplete(trade.BuyOrSellAndCompleteOpts{
		Market:      market,
		Amount:      uint64(amountInSatoshi),
		Asset:       market.BaseAsset,
		PrivateKey:  privkey,
		BlindingKey: blindkey,
	})
	if err != nil {
		log.Fatal(err)
	}
	elapsed = time.Since(start)
	fmt.Printf("TRADE TOOK %fs\n", elapsed.Seconds())
	fmt.Printf("bought %.2f LBTC in tx %s\n", amountInBtc, txid)
}

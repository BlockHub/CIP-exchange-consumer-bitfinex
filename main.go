package main

import (
	"fmt"
	//"log"
	"github.com/bitfinexcom/bitfinex-api-go/v1"
	"log"
	"strings"
	"CIP-exchange-consumer/pkg/handlers"
	"CIP-exchange-consumer/pkg/consumer"
	"github.com/jinzhu/gorm"
	 _ "github.com/jinzhu/gorm/dialects/postgres"

	"os"
	db "CIP-exchange-consumer/internal/db"
	"time"
	"github.com/joho/godotenv"
)


func main() {
	// this loads all the constants stored in the .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}


	c := bitfinex.NewClient()

	pairs, err := c.Pairs.All()
	if nil != err {
		fmt.Println(err)
	}

	err = c.WebSocket.Connect()
	if err != nil {
		log.Fatal("Error connecting to web socket : ", err)
	}
	defer c.WebSocket.Close()

	gormdb, err := gorm.Open(os.Getenv("DB"), os.Getenv("DB_URL"))
	if err != nil{
		panic(err)
	}
	defer gormdb.Close()

	// migrations are only performed by GORM if a table/column/index does not exist.
	gormdb.AutoMigrate(&db.BitfinexMarket{}, &db.BitfinexTicker{}, &db.BitfinexOrder{}, &db.BitfinexOrderBook{})

	for _, pair := range pairs {
		market := db.BitfinexMarket{0, pair[0:3], pair[len(pair)-3:]}
		gormdb.Create(&market)
		orderbook := db.BitfinexOrderBook{0, market.ID, int64(time.Now().Unix())}
		gormdb.Create(&orderbook)

		bookChannel := make(chan []float64)
		trades_chan := make(chan []float64)

		c.WebSocket.AddSubscribe(bitfinex.ChanBook, strings.ToUpper(pair), bookChannel)
		c.WebSocket.AddSubscribe(bitfinex.ChanTrade, bitfinex.BTCUSD, trades_chan)

		orderhandler := handlers.OrderDbHandler{gormdb, orderbook}
		tickerhandler := handlers.PrintHandler{}
		go consumer.Consumer(bookChannel, orderhandler)
		go consumer.Consumer(trades_chan, tickerhandler)
	}

	err = c.WebSocket.Subscribe()
	if err != nil {
		log.Fatal(err)
	}
}



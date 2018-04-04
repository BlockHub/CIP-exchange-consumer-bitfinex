package pushers

import (
	"github.com/jinzhu/gorm"
	"log"
	"CIP-exchange-consumer-bitfinex/internal/db"
	"fmt"
	"time"
	"strings"
)



type Replicator struct {
	// local db
	Local gorm.DB

	//remote DB (the data warehouse)
	Remote gorm.DB

	//schema related settings

	//replication related settings
	Limit int64	// max rows to be fetched from remote and inserted (should be as high as possible)

}
// send the initial Markets data to remote
func (r *Replicator) PushMarkets() {
	markets := []db.BitfinexMarket{}
	r.Local.Limit(r.Limit).Find(&markets)

	// we don't delete the local copies of the markets, as they are needed for FK relations
	// and don't take up much space
	for _, market := range markets {
		err := r.Remote.Create(&market).Error
		if err != nil {
			if ! strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				log.Panic(err)
			}
		}
	}
}


// copy the markets table (should only be done once in a while, as new markets
// are only added once every few months.
func(r *Replicator) Start(){
	for true {
		fmt.Println("replicating")
		r.Replicate_ticker()
	}
}

// copy the ticker data from a chunk
func (r *Replicator) Replicate_ticker() {
	// an out interface to store lots of Order objects
	backup := r.Remote.Begin()
	local := r.Local.Begin()

	orders := []db.BitfinexOrder{}
	tickers := []db.BitfinexTicker{}
	books := []db.BitfinexOrderBook{}



	r.Local.Limit(r.Limit).Find(&orders)
	r.Local.Limit(r.Limit).Find(&tickers)
	r.Local.Limit(r.Limit).Order("time asc").Find(&books)


	if (len(orders) == 0) || (len(tickers) == 0){
		time.Sleep(10* time.Second)
		return
	}

	for i, book := range books	{
		if i == len(books) - 1 { break }
		err := backup.Create(&book).Error
		if err != nil{
			panic(err)
		}
		err = local.Delete(&book).Error
		if err != nil{
			panic(err)
		}
	}


	for _, order := range orders {
		err := backup.Create(&order).Error
		if err != nil{
			panic(err)
		}
		err = local.Delete(&order).Error
		if err != nil{
			panic(err)
		}
	}

	for _, ticker := range tickers {
		err := backup.Create(&ticker).Error
		if err != nil{
			panic(err)
		}
		err = local.Delete(&ticker).Error
		if err != nil{
			panic(err)
		}
	}

	err := backup.Commit().Error
	if err != nil{
		local.Rollback()
		backup.Rollback()
		log.Panic(err)
	}

	err = local.Commit().Error
	if err != nil{
		local.Rollback()
		log.Panic(err)
	}
}
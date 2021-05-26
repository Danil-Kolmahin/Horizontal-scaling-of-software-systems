package main

import (
	"github.com/KolmaginDanil/Horizontal-scaling-of-software-systems/datastore"
)

func main()  {
	db, err := datastore.NewDb("./storage")
	if err != nil {
		panic(err)
	}
	err = db.Put("key6", "value61")
	if err != nil {
		panic(err)
	}
}

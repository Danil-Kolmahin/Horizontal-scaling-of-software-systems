package main

import (
	"github.com/KolmaginDanil/Horizontal-scaling-of-software-systems/datastore"
)

func main()  {
	db, err := datastore.NewDb("./storage")
	if err != nil {
		panic(err)
	}
	db.Put("key1", "value11") //segment-1
	db.Put("key2", "value21") //segment-1

	db.Put("key3", "value31") //segment-2
	db.Put("key1", "value12") //segment-1 after segment-2 (no segment-2)

	db.Put("key3", "value32") //segment-2 after segment-3 (no segment-3)
	db.Put("key2", "value22") //segment-1 after segment-3 (no segment-3)
	//segment-3 dead

	db.Put("key4", "value41") //current-data
}

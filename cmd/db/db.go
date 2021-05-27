package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/KolmaginDanil/Horizontal-scaling-of-software-systems/datastore"
	"net/http"
)

type getStringRes struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type getInt64Res struct {
	Key   string `json:"key"`
	Value int64  `json:"value"`
}

type postData struct {
	Value interface{} `json:"value"`
}

var port = flag.Int("port", 8085, "server port")

var db, globalError = datastore.NewDb("./")

func getApi(w http.ResponseWriter, req *http.Request) {
	//http://localhost:8080/db/?key=some;type=another
	if req.Method == "GET" {
		keys, okKey := req.URL.Query()["key"]
		types, okType := req.URL.Query()["type"]
		typeValue := ""

		if !okKey || len(keys[0]) < 1 {
			w.WriteHeader(400)
			w.Write([]byte("Url Param 'key' is missing"))
			return
		}
		if !okType || len(types[0]) < 1 {
			typeValue = "string"
		} else {
			typeValue = types[0]
		}

		key := keys[0]

		if typeValue == "string" {
			value, err := db.Get(key)
			if err != nil {
				w.WriteHeader(400)
				w.Write([]byte(err.Error()))
				return
			}
			res := &getStringRes{Key: key, Value: value}
			jsonData, err := json.Marshal(res)
			if err != nil {
				w.WriteHeader(400)
				w.Write([]byte(err.Error()))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(jsonData)
		} else if typeValue == "int64" {
			value, err := db.GetInt64(key)
			if err != nil {
				w.WriteHeader(400)
				w.Write([]byte(err.Error()))
				return
			}
			res := &getInt64Res{Key: key, Value: value}
			jsonData, err := json.Marshal(res)
			if err != nil {
				w.WriteHeader(400)
				w.Write([]byte(err.Error()))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(jsonData)
			return
		} else {
			w.Write([]byte("Wrong type value"))
		}
		return
	} else if req.Method == "POST" {
		keys, okKey := req.URL.Query()["key"]
		if !okKey || len(keys[0]) < 1 {
			w.WriteHeader(400)
			w.Write([]byte("Url Param 'key' is missing"))
			return
		}
		key := keys[0]
		var data postData
		err := json.NewDecoder(req.Body).Decode(&data)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte(err.Error()))
			return
		}
		typeReq := fmt.Sprintf("%T", data.Value)
		if typeReq == "string" {
			err := db.Put(key, data.Value.(string))
			if err != nil {
				w.WriteHeader(400)
				w.Write([]byte(err.Error()))
				return
			}
			w.Write([]byte("ok"))
		}else if typeReq == "float64" {
			valueF := data.Value.(float64)
			valueI := int64(valueF)
			err := db.PutInt64(key, valueI)
			if err != nil {
				w.WriteHeader(400)
				w.Write([]byte(err.Error()))
				return
			}
			w.Write([]byte("ok"))
		}else {
			w.WriteHeader(400)
			w.Write([]byte("Wrong type value"))
		}
		return
	}
	w.WriteHeader(404)
	w.Write([]byte("Page not found (wrong method)"))
}

func main() {
	if globalError != nil {
		panic(globalError)
		return
	}
	http.HandleFunc("/db/", getApi)
	http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
}

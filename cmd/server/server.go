package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/KolmaginDanil/Horizontal-scaling-of-software-systems/httptools"
	"github.com/KolmaginDanil/Horizontal-scaling-of-software-systems/signal"
)

var port = flag.Int("port", 8080, "server port")

const confResponseDelaySec = "CONF_RESPONSE_DELAY_SEC"
const confHealthFailure = "CONF_HEALTH_FAILURE"

func main() {
	var jsonStr = []byte(`{"value":"`+time.Now().Format(time.RFC3339)+`"}`)
	client := http.DefaultClient
	req, err := http.NewRequest("POST", "http://database:8085/db/?key=dwoescanoe", bytes.NewBuffer(jsonStr))
	if err != nil {
		panic(err)
	}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		panic(err)
	}
	h := new(http.ServeMux)

	h.HandleFunc("/health", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("content-type", "text/plain")
		if failConfig := os.Getenv(confHealthFailure); failConfig == "true" {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte("FAILURE"))
		} else {
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte("OK"))
		}
	})

	report := make(Report)

	h.HandleFunc("/api/v1/some-data/", func(w http.ResponseWriter, r *http.Request) {
		keys, okKey := r.URL.Query()["key"]
		types, okType := r.URL.Query()["type"]
		typeValue := ""
		if !okKey || len(keys[0]) < 1 {
			w.WriteHeader(400)
			w.Write([]byte("Url Param 'key' is missing"))
			return
		}
		if okType || len(types) != 0 {
			typeValue = ";type="+types[0]
		}

		url := "http://database:8085/db/?key="+keys[0]+typeValue

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			w.WriteHeader(500)
			fmt.Println("here")
			w.Write([]byte("Req error"))
			return
		}
		res, err := client.Do(req)
		if err != nil && res.StatusCode != 200 {
			w.WriteHeader(404)
			w.Write([]byte("Db error"))
			return
		}
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("server error"))
			return
		}
		log.Println(string(bodyBytes))
		if string(bodyBytes) == "record does not exist" {
			w.WriteHeader(404)
		}
		w.Write(bodyBytes)
	})

	h.Handle("/report", report)

	server := httptools.CreateServer(*port, h)
	server.Start()
	signal.WaitForTerminationSignal()
}

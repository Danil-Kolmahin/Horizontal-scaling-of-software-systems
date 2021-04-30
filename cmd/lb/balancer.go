package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/KolmaginDanil/Horizontal-scaling-of-software-systems/httptools"
	"github.com/KolmaginDanil/Horizontal-scaling-of-software-systems/signal"
)

var (
	port       = flag.Int("port", 8090, "load balancer port")
	timeoutSec = flag.Int("timeout-sec", 3, "request timeout time in seconds")
	https      = flag.Bool("https", false, "whether backends support HTTPs")

	traceEnabled = flag.Bool("trace", false, "whether to include tracing information into responses")
)

var (
	timeout     = time.Duration(*timeoutSec) * time.Second
	serversPool = []string{
		"server1:8080",
		"server2:8080",
		"server3:8080",
	}
)

func scheme() string {
	if *https {
		return "https"
	}
	return "http"
}

// if the server responds to the request and returns 200 then true, if not - false
func health(dst string) bool {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	req, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s://%s/health", scheme(), dst), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	if resp.StatusCode != http.StatusOK {
		return false
	}
	return true
}

// send data for one of the servers and after forward this data to client
func forward(dst string, rw http.ResponseWriter, r *http.Request, i int) error {
	ctx, _ := context.WithTimeout(r.Context(), timeout)
	fwdRequest := r.Clone(ctx)
	fwdRequest.RequestURI = ""
	fwdRequest.URL.Host = dst
	fwdRequest.URL.Scheme = scheme()
	fwdRequest.Host = dst
	fwdRequest.Header.Set("lb-author", "someAuthor")
	fwdRequest.Header.Set("lb-req-cnt", strconv.Itoa(i))

	resp, err := http.DefaultClient.Do(fwdRequest)
	if err == nil {
		for k, values := range resp.Header {
			for _, value := range values {
				rw.Header().Add(k, value)
			}
		}
		if *traceEnabled {
			rw.Header().Set("lb-from", dst)
		}
		log.Println("fwd", resp.StatusCode, resp.Request.URL)
		rw.WriteHeader(resp.StatusCode)
		defer resp.Body.Close()
		_, err := io.Copy(rw, resp.Body)
		if err != nil {
			log.Printf("Failed to write response: %s", err)
		}
		return nil
	} else {
		log.Printf("Failed to get response from %s: %s", dst, err)
		rw.WriteHeader(http.StatusServiceUnavailable)
		return err
	}
}

// when the same string is entered, the same number will be returned (for testing)
func hash(addr string) int {
	log.Println("row addr : " + addr)
	ip, _, _ := net.SplitHostPort(addr)
	ip = strings.Replace(ip, "[", "", -1)
	ip = strings.Replace(ip, "]", "", -1)
	ip = strings.Replace(ip, ":", "", -1)
	ip = strings.Replace(ip, ".", "", -1)
	res, _ := strconv.Atoi(ip)
	rand.Seed(int64(res))
	res = rand.Int()
	log.Println("result : " + strconv.Itoa(res))
	return res
}

//find a live server
func balance(serversHealth []bool, serverIndex int) (int, error) {
	serversLength := len(serversHealth)
	for i := serverIndex; i <= serverIndex+serversLength; i++ {
		if serversHealth[i%serversLength] {
			return i % len(serversHealth), nil
		}
	}
	return -1, errors.New("all servers is dead")
}

func main() {
	flag.Parse()
	counter := 0
	//all servers dead
	healthArray := make([]bool, len(serversPool))
	for i := range healthArray {
		healthArray[i] = health(serversPool[i])
	}

	//after 10 sec check and change servers health status
	for i, server := range serversPool {
		server := server
		i := i
		go func() {
			for range time.Tick(10 * time.Second) {
				healthNow := health(server)
				healthArray[i] = healthNow
				log.Println(server, healthNow)
			}
		}()
	}

	frontend := httptools.CreateServer(*port, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		counter++
		serversIndex := hash(r.RemoteAddr)
		serverNumber, err := balance(healthArray, serversIndex)
		if err != nil {
			log.Println(err)
		} else {
			forward(serversPool[serverNumber], rw, r, counter)
		}
	}))

	log.Println("Starting load balancer...")
	log.Printf("Tracing support enabled: %t", *traceEnabled)
	frontend.Start()
	signal.WaitForTerminationSignal()
}

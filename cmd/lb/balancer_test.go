package main

import (
	"encoding/json"
	gocheck "gopkg.in/check.v1"
	"net/http"
	"net/http/httptest"
	//"os/exec"
	"testing"
)

func Test(t *testing.T) { gocheck.TestingT(t) }

type MySuite struct{}

var _ = gocheck.Suite(&MySuite{})

func (s *MySuite) TestHash(c *gocheck.C) {
	//Arrange
	testCases := []struct {
		address      string
		expectedHash int
	}{
		{
			address:      "172.0.10.29:8039",
			expectedHash: 75735823885486378,
		},
		{
			address:      "192.168.0.101:8736",
			expectedHash: 4319878349005173251,
		},
		{
			address:      "127.0.0.1:8080",
			expectedHash: 42641755437267490,
		},
	}
	//Act
	for _, data := range testCases {
		hash := hash(data.address)
		//Assert
		c.Assert(hash, gocheck.Equals, data.expectedHash)
	}
}

func (s *MySuite) TestBalance(c *gocheck.C) {
	//Arrange
	healthArray := []bool{true, false, true}
	testCases := []struct {
		chosenServer   int
		expectedServer int
	}{
		{
			chosenServer:   0,
			expectedServer: 0,
		},
		{
			chosenServer:   1,
			expectedServer: 2,
		},
		{
			chosenServer:   2,
			expectedServer: 2,
		},
	}
	//Act
	for _, data := range testCases {
		serverIndex, err := balance(healthArray, data.chosenServer)
		//Assert
		if err != nil {
			c.Error(err)
		} else {
			c.Assert(serverIndex, gocheck.Equals, data.expectedServer)
		}
	}
}

func (s *MySuite) TestIntegration(c *gocheck.C) {
	//Arrange
	healthArray := []bool{true, false, true}
	testCases := []struct {
		address        string
		expectedServer int
	}{
		{
			address:        "172.0.10.29:8039",
			expectedServer: 2,
		},
		{
			address:        "192.168.0.101:8738",
			expectedServer: 0,
		},
		{
			address:        "127.0.0.1:8080",
			expectedServer: 2,
		},
	}
	//Act
	for _, data := range testCases {
		serverIndex, err := balance(healthArray, hash(data.address)%len(healthArray))
		//Assert
		if err != nil {
			c.Error(err)
		} else {
			c.Assert(serverIndex, gocheck.Equals, data.expectedServer)
		}
	}
}

func (s *MySuite) TestIntegration1(c *gocheck.C) {
	healthArray := []bool{true, false, false}
	serversPool := []string{
		"localhost:8080",
		"localhost:8081",
		"localhost:8082",
	}
	testCases := []struct {
		address string
		result  []string
	}{
		{
			address: "172.0.10.29:8039",
			result:  []string{"1", "2"},
		},
		{
			address: "192.168.0.101:8738",
			result:  []string{"1", "2"},
		},
		{
			address: "127.0.0.1:8080",
			result:  []string{"1", "2"},
		},
	}

	handler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		serversIndex := hash(r.RemoteAddr)
		serverNumber, err := balance(healthArray, serversIndex)
		if err == nil {
			forward(serversPool[serverNumber], rw, r, 666)
		}
	})
	//h := new(http.ServeMux)
	//h.HandleFunc("/api/v1/some-data", func(rw http.ResponseWriter, r *http.Request) {
	//	rw.Header().Set("content-type", "application/json")
	//	rw.WriteHeader(http.StatusOK)
	//	_ = json.NewEncoder(rw).Encode([]string{
	//		"1", "2",
	//	})
	//})
	//server := httptools.CreateServer(8090, h)
	//server.Start()
	//main.Smth()
	//time.Sleep(time.Second * 10)
	for _, tc := range testCases {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "http://localhost:8090/api/v1/some-data", nil)
		req.RemoteAddr = tc.address
		handler.ServeHTTP(rec, req)
		var dat []string
		if err := json.Unmarshal(rec.Body.Bytes(), &dat); err != nil {
			c.Error(err)
		}
		//fmt.Println(dat)
		//fmt.Println(tc.result)
		c.Assert(dat, gocheck.DeepEquals, tc.result)
	}
}

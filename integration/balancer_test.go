package integration

import (
	"fmt"
	gocheck "gopkg.in/check.v1"
	"net/http"
	"testing"
	"time"
)

const baseAddress = "http://balancer:8090"

var client = http.Client{
	Timeout: 3 * time.Second,
}

var serversPool = []string{
	"server1:8080",
	"server2:8080",
	"server3:8080",
}

func Test(t *testing.T) { gocheck.TestingT(t) }

type MySuite struct{}

var _ = gocheck.Suite(&MySuite{})

func (s *MySuite) TestBalancer(c *gocheck.C) {
	//Arrange
	testCases := []struct {
		address string
		server  string
	}{
		{
			address: "172.0.10.29:8121",
			server:  serversPool[2],
		},
		{
			address: "192.168.0.101:8736",
			server:  serversPool[2],
		},
		{
			address: "127.0.0.1:3005",
			server:  serversPool[0],
		},
	}

	//Act
	for _, data := range testCases {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/some-data", baseAddress), nil)
		if err != nil {
			c.Error(err)
		}
		req.RemoteAddr = data.address
		c.Logf("request from [%s]", req.RemoteAddr)
		res, err := client.Do(req)
		if err != nil {
			c.Error(err)
		}
		//Assert
		server := res.Header.Get("lb-from")
		c.Logf("response from [%s]", server)
		c.Assert(server, gocheck.Equals, data.server)
	}
}

func (s *MySuite) BenchmarkBalancer(c *gocheck.C) {
	for i := 0; i < c.N; i++ {
		_, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
		if err != nil {
			c.Error(err)
		}
	}
}

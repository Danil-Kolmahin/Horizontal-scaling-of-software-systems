package integration

import (
	"encoding/json"
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

func Test(t *testing.T) { gocheck.TestingT(t) }

type MySuite struct{}

var _ = gocheck.Suite(&MySuite{})

func (s *MySuite) TestBalancer(c *gocheck.C) {
	res1, err := client.Get(fmt.Sprintf("%s/api/v1/some-data/?key=dwoescanoe", baseAddress))
	if err != nil {
		c.Error(err)
	}
	c.Assert(res1.StatusCode, gocheck.Equals, http.StatusOK)

	res2, err := client.Get(fmt.Sprintf("%s/api/v1/some-data/?key=dwoescanoe", baseAddress))
	if err != nil {
		c.Error(err)
	}
	c.Assert(res2.StatusCode, gocheck.Equals, http.StatusOK)

	c.Assert(res1.Header.Get("lb-from"), gocheck.Equals, res2.Header.Get("lb-from"))

	type getStringRes struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	var data1, data2 getStringRes
	err = json.NewDecoder(res1.Body).Decode(&data1)
	if err != nil {
		c.Error(err)
	}
	err = json.NewDecoder(res2.Body).Decode(&data2)
	if err != nil {
		c.Error(err)
	}
	if data1.Value == "" {
		c.Error("Empty value")
	}
	if data1.Value != data2.Value {
		c.Error("Bad values")
	}

}

func (s *MySuite) BenchmarkBalancer(c *gocheck.C) {
	for i := 0; i < c.N; i++ {
		res, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
		if err != nil {
			c.Error(err)
		}
		c.Assert(res.StatusCode, gocheck.Equals, http.StatusOK)
	}
}

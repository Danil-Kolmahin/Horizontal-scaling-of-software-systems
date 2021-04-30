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

func Test(t *testing.T) { gocheck.TestingT(t) }

type MySuite struct{}

var _ = gocheck.Suite(&MySuite{})

func (s *MySuite) TestBalancer(c *gocheck.C) {
	res1, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
	if err != nil {
		c.Error(err)
	}
	c.Assert(res1.StatusCode, gocheck.Equals, http.StatusOK)

	res2, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
	if err != nil {
		c.Error(err)
	}
	c.Assert(res2.StatusCode, gocheck.Equals, http.StatusOK)

	c.Assert(res1.Header.Get("lb-from"), gocheck.Equals, res2.Header.Get("lb-from"))

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

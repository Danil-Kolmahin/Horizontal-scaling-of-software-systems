package main

import (
	gocheck "gopkg.in/check.v1"
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

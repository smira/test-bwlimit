package bandwidth_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
)

type NetSuite struct {
	suite.Suite

	fs *FakeServer
}

func (suite *NetSuite) SetupTest() {
	suite.fs = NewFakeServer(&suite.Suite, 500_000, 200_000)
}

func (suite *NetSuite) TearDownTest() {
	suite.fs.Shutdown()
}

func (suite *NetSuite) TestSingleConnection() {
	FakeClientAssertBandwidth(&suite.Suite, suite.fs, 200_000)
}

func (suite *NetSuite) TestTwoConnections() {
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			FakeClientAssertBandwidth(&suite.Suite, suite.fs, 200_000)
		}()
	}

	wg.Wait()
}

func (suite *NetSuite) TestFiveConnections() {
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			FakeClientAssertBandwidth(&suite.Suite, suite.fs, 100_000)
		}()
	}
	wg.Wait()
}

func TestNetSuite(t *testing.T) {
	suite.Run(t, new(NetSuite))
}

package bandwidth_test

import (
	"context"
	"io"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/smira/test-bwlimit/bandwidth"
	"github.com/stretchr/testify/suite"
)

type LimitedReaderSuite struct {
	suite.Suite
}

func (suite *LimitedReaderSuite) assertBandwidthN(r io.Reader, expectedBandwidth float64, N int64) {
	start := time.Now()
	var (
		n   int64
		err error
	)
	if N == 0 {
		n, err = io.Copy(ioutil.Discard, r)
	} else {
		n, err = io.CopyN(ioutil.Discard, r, N)
	}
	duration := time.Since(start)
	suite.Assert().NoError(err)

	bandwidth := float64(n) / duration.Seconds()
	suite.Assert().InEpsilon(expectedBandwidth, bandwidth, 0.05,
		"expected bandwidth %.2f, actual bandwidth %.2f", expectedBandwidth, bandwidth) // 5% error allowed
}

func (suite *LimitedReaderSuite) assertBandwidth(r io.Reader, expectedBandwidth float64) {
	suite.assertBandwidthN(r, expectedBandwidth, 0)
}

func (suite *LimitedReaderSuite) TestSingleLimit() {
	suite.assertBandwidth(
		bandwidth.NewLimitedReader(
			context.Background(),
			&FakeReader{200_000},
			bandwidth.NewLimit(100_000)),
		100_000)

	suite.assertBandwidth(
		bandwidth.NewLimitedReader(
			context.Background(),
			&FakeReader{200},
			bandwidth.NewLimit(50, bandwidth.BurstBytes(5))),
		50)

	suite.assertBandwidth(
		bandwidth.NewLimitedReader(
			context.Background(),
			&FakeReader{2_000_000},
			bandwidth.NewLimit(10*1024*1024)),
		10*1024*1024)
}

func (suite *LimitedReaderSuite) TestAdjustBandwidth() {
	limit := bandwidth.NewLimit(100_000)
	reader := bandwidth.NewLimitedReader(
		context.Background(),
		&FakeReader{100_000_000},
		limit,
	)

	suite.assertBandwidthN(
		reader,
		100_000,
		200_000)

	limit.SetBandwidth(200_000)

	suite.assertBandwidthN(
		reader,
		200_000,
		200_000)

	limit.SetBandwidth(50_000)

	suite.assertBandwidthN(
		reader,
		50_000,
		100_000)
}

func (suite *LimitedReaderSuite) TestMultipleLimits() {
	// smallest of the limits actually works

	suite.assertBandwidth(
		bandwidth.NewLimitedReader(
			context.Background(),
			&FakeReader{200_000},
			bandwidth.NewLimit(100_000),
			bandwidth.NewLimit(200_000)),
		100_000)

	suite.assertBandwidth(
		bandwidth.NewLimitedReader(
			context.Background(),
			&FakeReader{2_000_000},
			bandwidth.NewLimit(20*1024*1024),
			bandwidth.NewLimit(10*1024*1024)),
		10*1024*1024)
}

func (suite *LimitedReaderSuite) TestSharedLimit() {
	limit := bandwidth.NewLimit(100_000)

	var wg sync.WaitGroup

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			suite.assertBandwidth(
				bandwidth.NewLimitedReader(
					context.Background(),
					&FakeReader{300_000},
					limit,
				),
				100_000/3)
		}()
	}
	wg.Wait()
}

func (suite *LimitedReaderSuite) TestSharedMultipleLayerLimit1() {
	globalLimit := bandwidth.NewLimit(100_000)

	var wg sync.WaitGroup

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			suite.assertBandwidth(
				bandwidth.NewLimitedReader(
					context.Background(),
					&FakeReader{300_000},
					globalLimit,
					bandwidth.NewLimit(50_000),
				),
				100_000/3)
		}()
	}
	wg.Wait()
}

func (suite *LimitedReaderSuite) TestSharedMultipleLayerLimit2() {
	globalLimit := bandwidth.NewLimit(100_000)

	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			suite.assertBandwidth(
				bandwidth.NewLimitedReader(
					context.Background(),
					&FakeReader{200_000},
					globalLimit,
					bandwidth.NewLimit(10_000),
				),
				10_000)
		}()
	}
	wg.Wait()
}

func TestLimitedReaderSuite(t *testing.T) {
	suite.Run(t, new(LimitedReaderSuite))
}

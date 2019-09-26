package bandwidth_test

import (
	"bytes"
	"crypto/rand"
	"io"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/smira/test-bwlimit/bandwidth"
)

type LimitedWriterSuite struct {
	suite.Suite
}

func (suite *LimitedWriterSuite) assertBandwidthN(w io.Writer, expectedBandwidth float64, N uint64) {
	r := &FakeReader{Size: N}

	start := time.Now()
	n, err := io.Copy(w, r)
	duration := time.Since(start)
	suite.Assert().NoError(err)

	bandwidth := float64(n) / duration.Seconds()
	suite.Assert().InEpsilon(expectedBandwidth, bandwidth, 0.05,
		"expected bandwidth %.2f, actual bandwidth %.2f", expectedBandwidth, bandwidth) // 5% error allowed
}

func (suite *LimitedWriterSuite) TestContents() {
	r := io.LimitReader(rand.Reader, 10000)
	var srcBuf bytes.Buffer

	limit := bandwidth.NewLimit(100_000, bandwidth.BurstBytes(128))

	var dstBuf bytes.Buffer
	w := bandwidth.NewLimitedWriter(&dstBuf, limit)
	_, err := io.Copy(w, io.TeeReader(r, &srcBuf))
	suite.Require().NoError(err)

	suite.Require().Equal(srcBuf.Bytes(), dstBuf.Bytes())
}

func (suite *LimitedWriterSuite) TestSingleLimit() {
	suite.assertBandwidthN(
		bandwidth.NewLimitedWriter(
			ioutil.Discard,
			bandwidth.NewLimit(100_000)),
		100_000,
		200_000)

	suite.assertBandwidthN(
		bandwidth.NewLimitedWriter(
			ioutil.Discard,
			bandwidth.NewLimit(50, bandwidth.BurstBytes(5))),
		50,
		200)

	suite.assertBandwidthN(
		bandwidth.NewLimitedWriter(
			ioutil.Discard,
			bandwidth.NewLimit(10*1024*1024)),
		10*1024*1024,
		2_000_000)
}

func (suite *LimitedWriterSuite) TestAdjustBandwidth() {
	limit := bandwidth.NewLimit(100_000)
	writer := bandwidth.NewLimitedWriter(
		ioutil.Discard,
		limit,
	)

	suite.assertBandwidthN(
		writer,
		100_000,
		200_000)

	limit.SetBandwidth(200_000)

	suite.assertBandwidthN(
		writer,
		200_000,
		200_000)

	limit.SetBandwidth(50_000)

	suite.assertBandwidthN(
		writer,
		50_000,
		100_000)
}

func (suite *LimitedWriterSuite) TestMultipleLimits() {
	// smallest of the limits actually works

	suite.assertBandwidthN(
		bandwidth.NewLimitedWriter(
			ioutil.Discard,
			bandwidth.NewLimit(100_000),
			bandwidth.NewLimit(200_000)),
		100_000,
		200_000)

	suite.assertBandwidthN(
		bandwidth.NewLimitedWriter(
			ioutil.Discard,
			bandwidth.NewLimit(20*1024*1024),
			bandwidth.NewLimit(10*1024*1024)),
		10*1024*1024,
		2_000_000)
}

func (suite *LimitedWriterSuite) TestSharedLimit() {
	limit := bandwidth.NewLimit(100_000)

	var wg sync.WaitGroup

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			suite.assertBandwidthN(
				bandwidth.NewLimitedWriter(
					ioutil.Discard,
					limit,
				),
				100_000/3,
				300_000)
		}()
	}
	wg.Wait()
}

func (suite *LimitedWriterSuite) TestSharedMultipleLayerLimit1() {
	globalLimit := bandwidth.NewLimit(100_000)

	var wg sync.WaitGroup

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			suite.assertBandwidthN(
				bandwidth.NewLimitedWriter(
					ioutil.Discard,
					globalLimit,
					bandwidth.NewLimit(50_000),
				),
				100_000/3,
				300_000)
		}()
	}
	wg.Wait()
}

func (suite *LimitedWriterSuite) TestSharedMultipleLayerLimit2() {
	globalLimit := bandwidth.NewLimit(100_000)

	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			suite.assertBandwidthN(
				bandwidth.NewLimitedWriter(
					ioutil.Discard,
					globalLimit,
					bandwidth.NewLimit(10_000),
				),
				10_000,
				200_000)
		}()
	}
	wg.Wait()
}

func TestLimitedWriterSuite(t *testing.T) {
	suite.Run(t, new(LimitedWriterSuite))
}

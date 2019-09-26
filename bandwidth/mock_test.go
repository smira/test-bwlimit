package bandwidth_test

import (
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/smira/test-bwlimit/bandwidth"
)

type FakeReader struct {
	Size uint64
}

func (r *FakeReader) Read(p []byte) (n int, err error) {
	n = len(p)
	if uint64(n) > r.Size {
		n = int(r.Size)
	}
	r.Size -= uint64(n)
	p = p[:n]

	if n == 0 {
		err = io.EOF
	}

	return
}

type FakeServer struct {
	suite      *suite.Suite
	tmpDir     string
	socketPath string
	l          net.Listener
	shutdown   chan struct{}
}

func NewFakeServer(suite *suite.Suite, limitGlobal, limitPerConn int) *FakeServer {
	var err error

	fs := &FakeServer{
		suite:    suite,
		shutdown: make(chan struct{}),
	}
	fs.tmpDir, err = ioutil.TempDir("", "bwtest")
	suite.Require().NoError(err)

	fs.socketPath = filepath.Join(fs.tmpDir, "server.sock")

	fs.l, err = net.Listen("unix", fs.socketPath)
	suite.Require().NoError(err)

	fs.l = bandwidth.NewLimitedListener(fs.l, limitGlobal, limitPerConn)

	go fs.listenLoop()

	return fs
}

func (fs *FakeServer) listenLoop() {
	for {
		var (
			conn net.Conn
			err  error
		)

		ch := make(chan struct{})

		go func() {
			conn, err = fs.l.Accept()
			close(ch)
		}()

		select {
		case <-fs.shutdown:
			return
		case <-ch:
			fs.suite.Require().NoError(err)
			go fs.handler(conn)
		}
	}
}

func (fs *FakeServer) handler(conn net.Conn) {
	defer conn.Close()
	io.Copy(conn, &FakeReader{Size: 1_000_000})
}

func (fs *FakeServer) Shutdown() {
	fs.shutdown <- struct{}{}

	fs.suite.Require().NoError(fs.l.Close())

	fs.suite.Require().NoError(os.RemoveAll(fs.tmpDir))
}

func FakeClientAssertBandwidth(suite *suite.Suite, fs *FakeServer, expectedBandwidth float64) {
	conn, err := net.Dial("unix", fs.socketPath)
	suite.Require().NoError(err)

	defer conn.Close()

	start := time.Now()
	n, err := io.Copy(ioutil.Discard, conn)
	bandwidth := float64(n) / time.Since(start).Seconds()
	suite.Assert().NoError(err)
	suite.Assert().InEpsilon(expectedBandwidth, bandwidth, 0.05,
		"expected bandwidth %.2f, actual bandwidth %.2f", expectedBandwidth, bandwidth) // 5% error allowed
}

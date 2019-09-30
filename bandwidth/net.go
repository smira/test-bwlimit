package bandwidth

import "net"

type wrappedConn struct {
	net.Conn
	limitedWriter *LimitedWriter
}

func (wrapped *wrappedConn) Write(p []byte) (int, error) {
	return wrapped.limitedWriter.Write(p)
}

type wrappedListener struct {
	net.Listener

	globalLimit  *Limit
	limitPerConn float64
}

func (wrapped *wrappedListener) Accept() (net.Conn, error) {
	conn, err := wrapped.Listener.Accept()
	if err != nil {
		return conn, err
	}

	return &wrappedConn{
		Conn:          conn,
		limitedWriter: NewLimitedWriter(conn, wrapped.globalLimit, NewLimit(wrapped.limitPerConn)),
	}, nil
}

// NewLimitedListener provides wrapper around net.Listener which wraps Conn's Write method
// to control bandwidth on global listener level and per-connection.
//
// Implementation is based on LimitedWriter.
// limitGlobal and limitPerConn are bandwidth limits in bytes/sec.
func NewLimitedListener(l net.Listener, limitGlobal, limitPerConn int) net.Listener {
	return &wrappedListener{
		Listener:     l,
		globalLimit:  NewLimit(float64(limitGlobal)),
		limitPerConn: float64(limitPerConn),
	}
}

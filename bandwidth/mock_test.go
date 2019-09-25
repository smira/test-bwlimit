package bandwidth_test

import (
	"io"
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

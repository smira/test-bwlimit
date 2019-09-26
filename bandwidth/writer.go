package bandwidth

import (
	"errors"
	"io"
	"time"
)

// LimitedWriter controls bandwidth consumed by the writer
type LimitedWriter struct {
	wrapped io.Writer
	limits  []*Limit
}

// NewLimitedWriter builds a Limiter wrapping io.Writer with specified limits
func NewLimitedWriter(wrap io.Writer, limits ...*Limit) *LimitedWriter {
	return &LimitedWriter{
		wrapped: wrap,
		limits:  limits,
	}
}

func (lim *LimitedWriter) Write(p []byte) (int, error) {
	minBurst := int(^(uint(0)) >> 1)

	for _, limit := range lim.limits {
		if limit.Burst() < minBurst {
			minBurst = limit.Burst()
		}
	}

	var (
		err error
		n   int
	)

	N := len(p)

	for n = 0; n < N; {
		var buf []byte

		if N > n+minBurst {
			buf = p[n : n+minBurst]
		} else {
			buf = p[n:]
		}

		var maxDelay time.Duration

		now := time.Now()
		for i := range lim.limits {
			reservation := lim.limits[i].ReserveN(now, len(buf))
			if !reservation.OK() {
				err = errors.New("reservation error") // should never happen
				return n, err
			}
			delay := reservation.DelayFrom(now)
			if delay > maxDelay {
				maxDelay = delay
			}
		}

		if maxDelay > 0 {
			time.Sleep(maxDelay)
		}

		var written int
		written, err = lim.wrapped.Write(buf)
		n += written

		if err != nil {
			return n, err
		}
	}

	return n, err
}

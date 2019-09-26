package bandwidth

import (
	"context"
	"errors"
	"io"
	"time"
)

// LimitedReader controls bandwidth consumed by the reader
type LimitedReader struct {
	ctx     context.Context
	wrapped io.Reader
	limits  []*Limit
}

// NewLimitedReader builds a Limiter wrapping io.Reader with specified limits
func NewLimitedReader(ctx context.Context, wrap io.Reader, limits ...*Limit) *LimitedReader {
	return &LimitedReader{
		ctx:     ctx,
		wrapped: wrap,
		limits:  limits,
	}
}

func (lim *LimitedReader) Read(p []byte) (int, error) {
	minBurst := int(^(uint(0)) >> 1)

	for _, limit := range lim.limits {
		if limit.Burst() < minBurst {
			minBurst = limit.Burst()
		}
	}

	if len(p) > minBurst {
		p = p[:minBurst]
	}

	n, err := lim.wrapped.Read(p)
	if n == 0 || len(lim.limits) == 0 {
		return n, err
	}

	var maxDelay time.Duration

	now := time.Now()
	for i := range lim.limits {
		reservation := lim.limits[i].ReserveN(now, n)
		if !reservation.OK() {
			err = errors.New("reservation error") // should never happen
			return n, err
		}
		delay := reservation.DelayFrom(now)
		if delay > maxDelay {
			maxDelay = delay
		}
	}

	if maxDelay <= 0 {
		return n, err
	}

	timer := time.NewTimer(maxDelay)
	defer timer.Stop()

	select {
	case <-timer.C:
	case <-lim.ctx.Done():
		err = lim.ctx.Err()
	}

	return n, err
}

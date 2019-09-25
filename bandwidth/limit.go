package bandwidth

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Default burst size matches default io.Copy buffer.
const defaultBurst = 8192

// Limit describes bandwidth limit for "token bucket".
//
// There are two fundamental parameters:
//  * rate controls bandwidth limit (bytes/sec)
//  * burst controls bucket size (bytes)
//
// Burst should be larger than regular packet (buffer) size.
//
// Burst can be given as number of bytes or as time interval,
// in the latter case burst is calculated as (rate * interval).
type Limit struct {
	mu            sync.RWMutex
	limiter       *rate.Limiter
	rate          float64
	burst         int
	burstInterval time.Duration
}

// LimitOption controls Limit parameters.
type LimitOption func(*Limit)

// BurstInterval sets burst interval as interval, final burst in bytes
// is calculated as interval*rate.
func BurstInterval(interval time.Duration) LimitOption {
	return func(limit *Limit) {
		limit.burstInterval = interval
	}
}

// BurstBytes sets burst as number of bytes.
func BurstBytes(bytes int) LimitOption {
	return func(limit *Limit) {
		limit.burst = bytes
	}
}

// NewLimit builds new Limit with specified options.
//
// Bandwidth is bandwidth limit in bytes/sec.
func NewLimit(bandwidth float64, options ...LimitOption) *Limit {
	limit := &Limit{
		rate:  bandwidth,
		burst: defaultBurst,
	}

	for _, option := range options {
		option(limit)
	}

	if limit.burstInterval != 0 {
		limit.burst = int(limit.burstInterval.Seconds() * limit.rate)
	}

	if limit.burst == 0 {
		limit.burst = defaultBurst
	}

	limit.limiter = rate.NewLimiter(rate.Limit(limit.rate), limit.burst)

	return limit
}

// Burst returns current burst value.
func (limit *Limit) Burst() int {
	limit.mu.RLock()
	defer limit.mu.RUnlock()

	return limit.burst
}

// ReserveN wraps rate.Limiter.ReserveN
func (limit *Limit) ReserveN(now time.Time, n int) *rate.Reservation {
	limit.mu.RLock()
	defer limit.mu.RUnlock()

	return limit.limiter.ReserveN(now, n)
}

// SetBandwidth adjusts bandwidth limit for the running limit
func (limit *Limit) SetBandwidth(bandwidth float64) {
	limit.mu.Lock()
	defer limit.mu.Unlock()

	limit.rate = bandwidth
	limit.limiter.SetLimit(rate.Limit(limit.rate))

	if limit.burstInterval != 0 {
		limit.burst = int(limit.burstInterval.Seconds() * limit.rate)
		limit.limiter.SetBurst(limit.burst)
	}
}

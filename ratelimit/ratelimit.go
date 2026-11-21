package ratelimit

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ngamux/ngamux"
)

const (
	HeaderLimit      = "X-RateLimit-Limit"
	HeaderRemaining  = "X-RateLimit-Remaining"
	HeaderReset      = "X-RateLimit-Reset"
	HeaderRetryAfter = "Retry-After"
)

type bucket struct {
	tokens float64
	last   time.Time
}

type decision struct {
	allowed    bool
	remaining  int
	retryAfter time.Duration
	resetAt    time.Time
}

type limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	config  Config
	lastGC  time.Time
	now     func() time.Time
}

func newLimiter(config Config) *limiter {
	now := time.Now
	return &limiter{
		buckets: make(map[string]*bucket),
		config:  config,
		lastGC:  now(),
		now:     now,
	}
}

func (l *limiter) durationFor(tokens float64) time.Duration {
	rate := float64(l.config.Requests) / float64(l.config.Window)
	return time.Duration(math.Ceil(tokens / rate))
}

func (l *limiter) allow(key string) decision {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.cleanup(now)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: float64(l.config.Burst), last: now}
		l.buckets[key] = b
	}

	rate := float64(l.config.Requests) / float64(l.config.Window)
	if elapsed := now.Sub(b.last); elapsed > 0 {
		b.tokens = math.Min(float64(l.config.Burst), b.tokens+float64(elapsed)*rate)
		b.last = now
	}

	if b.tokens >= 1 {
		b.tokens--
		return decision{
			allowed:    true,
			remaining:  int(b.tokens),
			retryAfter: 0,
			resetAt:    now.Add(l.durationFor(float64(l.config.Burst) - b.tokens)),
		}
	}

	retryAfter := l.durationFor(1 - b.tokens)
	return decision{
		allowed:    false,
		remaining:  0,
		retryAfter: retryAfter,
		resetAt:    now.Add(retryAfter),
	}
}

func (l *limiter) cleanup(now time.Time) {
	if now.Sub(l.lastGC) < l.config.Window {
		return
	}

	l.lastGC = now
	for key, b := range l.buckets {
		if now.Sub(b.last) >= l.config.Window {
			delete(l.buckets, key)
		}
	}
}

func defaultErrorHandler(rw http.ResponseWriter, _ *http.Request, retryAfter time.Duration) {
	seconds := int(math.Ceil(retryAfter.Seconds()))
	if seconds < 1 {
		seconds = 1
	}

	rw.Header().Set(HeaderRetryAfter, strconv.Itoa(seconds))
	ngamux.Res(rw).Status(http.StatusTooManyRequests).JSON(ngamux.Map{
		"error": "too many requests",
	})
}

func New(configs ...Config) ngamux.MiddlewareFunc {
	var config Config
	if len(configs) > 0 {
		config = configs[0]
	}
	config = buildConfig(config)
	l := newLimiter(config)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(rw http.ResponseWriter, r *http.Request) {
			d := l.allow(config.KeyFunc(r))

			rw.Header().Set(HeaderLimit, strconv.Itoa(config.Requests))
			rw.Header().Set(HeaderRemaining, strconv.Itoa(d.remaining))
			rw.Header().Set(HeaderReset, strconv.FormatInt(d.resetAt.Unix(), 10))

			if !d.allowed {
				config.ErrorHandler(rw, r, d.retryAfter)
				return
			}

			next(rw, r)
		}
	}
}

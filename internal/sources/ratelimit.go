package sources

import (
	"sync"
	"time"
)

type RateLimiter struct {
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
	mu         sync.Mutex
}

func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (r *RateLimiter) Wait() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.refill()

	for r.tokens <= 0 {
		r.mu.Unlock()
		time.Sleep(100 * time.Millisecond)
		r.mu.Lock()
		r.refill()
	}

	r.tokens--
}

func (r *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(r.lastRefill)

	if elapsed >= r.refillRate {
		tokensToAdd := int(elapsed / r.refillRate)
		r.tokens += tokensToAdd

		if r.tokens > r.maxTokens {
			r.tokens = r.maxTokens
		}

		r.lastRefill = now
	}
}

func (r *RateLimiter) TryAcquire() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.refill()

	if r.tokens > 0 {
		r.tokens--
		return true
	}

	return false
}

var (
	githubRateLimiter   = NewRateLimiter(60, time.Minute)
	modrinthRateLimiter = NewRateLimiter(300, time.Minute)
)

func GetGitHubRateLimiter() *RateLimiter {
	return githubRateLimiter
}

func GetModrinthRateLimiter() *RateLimiter {
	return modrinthRateLimiter
}

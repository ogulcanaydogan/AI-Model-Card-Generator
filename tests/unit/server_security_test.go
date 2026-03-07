package unit

import (
	"testing"
	"time"

	"github.com/yapay/ai-model-card-generator/pkg/server"
)

func TestParseAPIKeys(t *testing.T) {
	keys := server.ParseAPIKeys(" alpha, beta ,, gamma ")
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}
	for _, key := range []string{"alpha", "beta", "gamma"} {
		if _, ok := keys[key]; !ok {
			t.Fatalf("missing parsed key: %s", key)
		}
	}
}

func TestRateLimiterAllowDenyAndRefill(t *testing.T) {
	limiter := server.NewRateLimiter(60, 1) // 1 token/sec, burst 1
	now := time.Now().UTC()

	if !limiter.Allow("client|/generate", now) {
		t.Fatalf("first request should be allowed")
	}
	if limiter.Allow("client|/generate", now) {
		t.Fatalf("second immediate request should be denied")
	}
	if !limiter.Allow("client|/generate", now.Add(time.Second)) {
		t.Fatalf("request after refill window should be allowed")
	}
}

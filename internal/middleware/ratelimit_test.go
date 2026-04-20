package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimit_AllowsBurstThenBlocks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit(3, time.Minute))
	r.GET("/r", func(c *gin.Context) { c.Status(http.StatusOK) })

	do := func() int {
		req := httptest.NewRequest(http.MethodGet, "/r", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	for i := 0; i < 3; i++ {
		if code := do(); code != http.StatusOK {
			t.Fatalf("request %d: code = %d, want 200", i+1, code)
		}
	}
	if code := do(); code != http.StatusTooManyRequests {
		t.Fatalf("4th request: code = %d, want 429", code)
	}
}

func TestRateLimit_ResetsAfterWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit(1, 20*time.Millisecond))
	r.GET("/r", func(c *gin.Context) { c.Status(http.StatusOK) })

	do := func() int {
		req := httptest.NewRequest(http.MethodGet, "/r", nil)
		req.RemoteAddr = "2.3.4.5:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	if code := do(); code != http.StatusOK {
		t.Fatalf("first: code = %d, want 200", code)
	}
	if code := do(); code != http.StatusTooManyRequests {
		t.Fatalf("second (same window): code = %d, want 429", code)
	}

	time.Sleep(30 * time.Millisecond)
	if code := do(); code != http.StatusOK {
		t.Fatalf("after window: code = %d, want 200", code)
	}
}

func TestRateLimit_IsPerIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit(1, time.Minute))
	r.GET("/r", func(c *gin.Context) { c.Status(http.StatusOK) })

	do := func(ip string) int {
		req := httptest.NewRequest(http.MethodGet, "/r", nil)
		req.RemoteAddr = ip + ":1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	if code := do("9.9.9.1"); code != http.StatusOK {
		t.Fatalf("ip1: code = %d, want 200", code)
	}
	if code := do("9.9.9.2"); code != http.StatusOK {
		t.Fatalf("ip2: code = %d, want 200", code)
	}
	if code := do("9.9.9.1"); code != http.StatusTooManyRequests {
		t.Fatalf("ip1 second: code = %d, want 429", code)
	}
}

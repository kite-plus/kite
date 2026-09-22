package auth

import (
	"fmt"
	"sync"
	"time"
)

// Throttle slows down repeated failures against a secret.
//
// It is a type rather than three fields on the guard because sign-in is not
// the only door: the first-run setup token is checked before any account
// exists, and a second hand-rolled counter would be a second place for the
// backoff to be subtly wrong.
type Throttle struct {
	free int

	mu       sync.Mutex
	failures int
	blocked  time.Time
}

// Attempts are free until there have been this many in a row, after which
// each one costs the next a doubling delay. Five is well past a typo and far
// short of anything a person does deliberately.
const freeAttempts = 5

const maxBackoff = 5 * time.Minute

// TooManyAttempts reports an attempt refused because of earlier failures. It
// carries how long is left, which the client shows rather than guessing.
type TooManyAttempts struct{ RetryAfter time.Duration }

func (e *TooManyAttempts) Error() string {
	return fmt.Sprintf("auth: too many attempts; try again in %s", e.RetryAfter.Round(time.Second))
}

// NewThrottle returns a throttle that allows free attempts in a row before it
// starts making the caller wait.
func NewThrottle(free int) *Throttle { return &Throttle{free: free} }

// Wait reports how long a refused attempt has left, or zero when one may
// proceed.
func (t *Throttle) Wait(now time.Time) time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	if now.Before(t.blocked) {
		return t.blocked.Sub(now)
	}
	return 0
}

// Refused reports a failed attempt, which is what lengthens the next wait.
func (t *Throttle) Refused(now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.failures++
	if over := t.failures - t.free; over > 0 {
		delay := time.Second << min(over-1, 10)
		t.blocked = now.Add(min(delay, maxBackoff))
	}
}

// Accepted clears the record, so that one success costs the next attempt
// nothing.
func (t *Throttle) Accepted() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.failures, t.blocked = 0, time.Time{}
}

// Check is [Wait] as an error, for a caller that would only turn it into one.
func (t *Throttle) Check(now time.Time) error {
	if wait := t.Wait(now); wait > 0 {
		return &TooManyAttempts{RetryAfter: wait}
	}
	return nil
}

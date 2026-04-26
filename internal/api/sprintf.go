package api

import "fmt"

// fmtSprintf isolates the dependency on the fmt package — kept in its own
// file so callers don't accidentally pull fmt into envelope.go's hot path.
func fmtSprintf(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}

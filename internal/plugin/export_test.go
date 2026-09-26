package plugin

import "time"

// SetPageLimit shortens how long a module may take over a page, for a test
// of what happens when one takes too long, and returns what undoes it.
func SetPageLimit(d time.Duration) func() {
	was := pageLimit
	pageLimit = d
	return func() { pageLimit = was }
}

// Package perf measures Kite against the latency targets its design sets for
// each milestone, on a site of the size those targets name.
//
// The measurements are tests, skipped unless KITE_PERF is set, because they
// generate thousands of files and time real servers: `make perf` runs them.
// A target missed fails the test with the numbers, so a regression is found
// by the change that caused it rather than by an author with a large site.
package perf

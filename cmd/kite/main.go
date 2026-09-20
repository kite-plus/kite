// Command kite is the Kite command line interface.
package main

import (
	"fmt"
	"os"

	"github.com/kite-plus/kite/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "kite:", err)
		os.Exit(1)
	}
}

// cmd/race-results/main.go
package main

import (
	"fmt"
	"os"

	"github.com/jiahongchen/race-results/internal/cli"
	"github.com/jiahongchen/race-results/internal/provider"
)

var version = "0.1.0"

func main() {
	reg := provider.NewRegistry()
	// Adapters register here as Phase 2 lands them.
	root := cli.NewRoot(reg)
	root.Version = version
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

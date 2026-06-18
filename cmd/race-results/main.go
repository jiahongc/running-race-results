// cmd/race-results/main.go
package main

import (
	"fmt"
	"os"

	"github.com/jiahongchen/race-results/internal/cli"
	"github.com/jiahongchen/race-results/internal/provider"
	"github.com/jiahongchen/race-results/internal/provider/athlinks"
	"github.com/jiahongchen/race-results/internal/provider/mika"
	"github.com/jiahongchen/race-results/internal/provider/nyrr"
	"github.com/jiahongchen/race-results/internal/provider/raceresult"
)

var version = "0.1.0"

func main() {
	reg := provider.NewRegistry()
	reg.Register(nyrr.New())
	reg.Register(mika.New())
	reg.Register(athlinks.New())
	reg.Register(raceresult.New())
	root := cli.NewRoot(reg)
	root.Version = version
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

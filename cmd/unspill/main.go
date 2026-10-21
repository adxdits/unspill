package main

import (
	"os"

	"github.com/adxdits/unspill/internal/cli"
)

var version = "0.2.0-dev"

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, version))
}

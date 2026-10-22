package main

import (
	"os"

	"github.com/adxdits/unspill/internal/cli"
	"github.com/adxdits/unspill/scanner"
)

var version = scanner.Version

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, version))
}

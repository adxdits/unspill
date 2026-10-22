package scanner_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/adxdits/unspill/scanner"
)

func ExampleScan() {
	report := scanner.Scan(context.Background(), scanner.Options{
		Paths: []string{"../testdata/demo"},
	})
	if err := report.Err(); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%d files, %d findings\n", report.Coverage.FilesScanned, len(report.Findings))
	// Output: 4 files, 4 findings
}

func ExampleReport_Err() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	report := scanner.Scan(ctx, scanner.Options{Paths: []string{"../testdata/demo"}})
	var issue scanner.ScanError
	if errors.As(report.Err(), &issue) {
		fmt.Println(issue.Code)
	}
	// Output: cancelled
}

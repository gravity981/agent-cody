//go:build !linux

package main

import (
	"fmt"
	"os"
)

func runRelay([]string) int {
	fmt.Fprintln(os.Stderr, "cody-bridge relay is only supported on Linux")
	return 2
}

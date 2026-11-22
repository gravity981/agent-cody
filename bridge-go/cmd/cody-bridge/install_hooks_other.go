//go:build !linux

package main

import (
	"fmt"
	"os"
)

func runHookInstaller([]string) int {
	fmt.Fprintln(os.Stderr, "cody-bridge install-hooks is only available in the WSL relay binary")
	return 2
}

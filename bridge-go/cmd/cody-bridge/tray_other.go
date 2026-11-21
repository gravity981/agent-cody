//go:build !windows

package main

func runWindowsTray(string) (int, bool) {
	return 0, false
}

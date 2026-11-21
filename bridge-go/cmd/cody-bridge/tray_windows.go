//go:build windows

package main

import "fyne.io/systray"

func runWindowsTray(onReady, onExit func()) {
	systray.Run(onReady, onExit)
}

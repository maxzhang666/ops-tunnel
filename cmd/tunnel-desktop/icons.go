package main

import (
	_ "embed"
	"runtime"
)

//go:embed tray/mac_gray.png
var macGray []byte

//go:embed tray/mac_yellow.png
var macYellow []byte

//go:embed tray/mac_empty.png
var macEmpty []byte

//go:embed tray/win_gray.png
var winGray []byte

//go:embed tray/win_yellow.png
var winYellow []byte

//go:embed tray/win_empty.png
var winEmpty []byte

var (
	iconGray   []byte
	iconYellow []byte
	iconEmpty  []byte
)

func init() {
	if runtime.GOOS == "darwin" {
		iconGray = macGray
		iconYellow = macYellow
		iconEmpty = macEmpty
	} else {
		iconGray = winGray
		iconYellow = winYellow
		iconEmpty = winEmpty
	}
}

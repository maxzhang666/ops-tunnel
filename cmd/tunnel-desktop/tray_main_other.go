//go:build !darwin

package main

func runOnMainThread(fn func()) {
	fn()
}

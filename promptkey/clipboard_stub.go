//go:build !windows

package main

func captureSelectedText() (string, bool) { return "", false }

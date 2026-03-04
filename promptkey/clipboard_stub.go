//go:build !windows

package main

func captureSelectedText(_ bool) (string, bool) { return "", false }

//go:build !windows

package main

func platformCaptureContext(_ ContextConfig) Context { return Context{} }

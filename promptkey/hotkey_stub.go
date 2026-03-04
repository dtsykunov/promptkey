//go:build !windows

package main

func (a *App) startHotkey(_ func())                {}
func (a *App) resetHotkey()                        {}
func (a *App) startFocusWatcher(_ <-chan struct{}) {}
func getCursorPos() (int, int)                     { return 0, 0 }

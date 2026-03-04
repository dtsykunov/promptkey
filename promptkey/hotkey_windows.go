//go:build windows

package main

import (
	"runtime"
	"syscall"
	"unsafe"
)

const (
	modControl  = 0x0002
	modAlt      = 0x0001
	modNoRepeat = 0x4000
	vkGrave     = 0xC0 // VK_OEM_3 — the ` / ~ key
	wmHotkey    = 0x0312
)

var (
	user32          = syscall.NewLazyDLL("user32.dll")
	pRegisterHotKey = user32.NewProc("RegisterHotKey")
	pGetMessage     = user32.NewProc("GetMessageW")
	pGetCursorPos   = user32.NewProc("GetCursorPos")
)

type winPoint struct{ X, Y int32 }

// winMsg matches the Win32 MSG struct layout.
type winMsg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      winPoint
	_       uint32 // padding
}

func (a *App) startHotkey(cb func()) {
	go func() {
		runtime.LockOSThread()
		r, _, _ := pRegisterHotKey.Call(0, 1, modControl|modAlt|modNoRepeat, vkGrave)
		debugf("startHotkey: RegisterHotKey returned %d (0=failed)", r)
		var msg winMsg
		for {
			r, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if r == 0 {
				break // WM_QUIT
			}
			if msg.Message == wmHotkey {
				debugf("startHotkey: WM_HOTKEY fired")
				cb()
			}
		}
	}()
}

func getCursorPos() (int, int) {
	var pt winPoint
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	return int(pt.X), int(pt.Y)
}

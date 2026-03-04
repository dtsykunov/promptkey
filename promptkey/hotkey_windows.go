//go:build windows

package main

import (
	"runtime"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

const (
	modNoRepeat = 0x4000
	wmHotkey    = 0x0312
	wmQuit      = 0x0012
)

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	pRegisterHotKey     = user32.NewProc("RegisterHotKey")
	pUnregisterHotKey   = user32.NewProc("UnregisterHotKey")
	pGetMessage         = user32.NewProc("GetMessageW")
	pGetCursorPos       = user32.NewProc("GetCursorPos")
	pPostThreadMessageW = user32.NewProc("PostThreadMessageW")
	pGetCurrentThreadId = kernel32.NewProc("GetCurrentThreadId")
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
		tid, _, _ := pGetCurrentThreadId.Call()
		atomic.StoreUint32(&a.hotkeyThreadID, uint32(tid))

		key := a.cfg.Hotkey
		if key == "" {
			key = defaultHotkey
		}
		mods, vk, err := parseHotkey(key)
		if err != nil {
			debugf("bad hotkey %q: %v", key, err)
			return
		}

		r, _, _ := pRegisterHotKey.Call(0, 1, uintptr(mods|modNoRepeat), uintptr(vk))
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
		pUnregisterHotKey.Call(0, 1)
	}()
}

func (a *App) resetHotkey() {
	tid := atomic.LoadUint32(&a.hotkeyThreadID)
	if tid != 0 {
		pPostThreadMessageW.Call(uintptr(tid), wmQuit, 0, 0)
		time.Sleep(150 * time.Millisecond)
	}
	a.startHotkey(a.showPopup)
}

func getCursorPos() (int, int) {
	var pt winPoint
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	return int(pt.X), int(pt.Y)
}

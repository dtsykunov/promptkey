//go:build windows

package main

import (
	"syscall"
	"time"
	"unsafe"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	pGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	pFindWindowW         = user32.NewProc("FindWindowW")
)

func (a *App) startFocusWatcher(stopCh <-chan struct{}) {
	go func() {
		title, _ := syscall.UTF16PtrFromString("promptkey")
		hwnd, _, _ := pFindWindowW.Call(0, uintptr(unsafe.Pointer(title)))
		if hwnd == 0 {
			return
		}
		// Wait for the popup to become the foreground window (up to 500 ms).
		for i := 0; i < 10; i++ {
			select {
			case <-stopCh:
				return
			case <-time.After(50 * time.Millisecond):
			}
			fg, _, _ := pGetForegroundWindow.Call()
			if fg == hwnd {
				break
			}
		}
		// Poll until a different window takes the foreground.
		for {
			select {
			case <-stopCh:
				return
			case <-time.After(80 * time.Millisecond):
			}
			fg, _, _ := pGetForegroundWindow.Call()
			if fg != hwnd {
				wailsruntime.WindowHide(a.ctx)
				wailsruntime.EventsEmit(a.ctx, "popup:dismiss")
				return
			}
		}
	}()
}

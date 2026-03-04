//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	ntdll                     = syscall.NewLazyDLL("ntdll.dll")
	pOpenClipboard            = user32.NewProc("OpenClipboard")
	pCloseClipboard           = user32.NewProc("CloseClipboard")
	pGetClipboardData         = user32.NewProc("GetClipboardData")
	pGlobalLock               = kernel32.NewProc("GlobalLock")
	pGlobalUnlock             = kernel32.NewProc("GlobalUnlock")
	pGlobalSize               = kernel32.NewProc("GlobalSize")
	pRtlMoveMemory            = kernel32.NewProc("RtlMoveMemory")
	pGetWindowTextW           = user32.NewProc("GetWindowTextW")
	pGetWindowTextLengthW     = user32.NewProc("GetWindowTextLengthW")
	pRtlGetVersion            = ntdll.NewProc("RtlGetVersion")
	pGetUserDefaultLocaleName = kernel32.NewProc("GetUserDefaultLocaleName")
)

// rtlOsVersionInfoW matches the RTL_OSVERSIONINFOW struct.
type rtlOsVersionInfoW struct {
	OSVersionInfoSize uint32
	MajorVersion      uint32
	MinorVersion      uint32
	BuildNumber       uint32
	PlatformId        uint32
	CSDVersion        [128]uint16
}

func platformCaptureContext(cfg ContextConfig) Context {
	var ctx Context
	if cfg.Clipboard {
		ctx.Clipboard = readClipboard()
	}
	if cfg.ActiveApp {
		ctx.App = readForegroundWindowTitle()
	}
	if cfg.OSLocale {
		ctx.OS = readOSVersion()
		ctx.Locale = readLocale()
	}
	return ctx
}

func readClipboard() string {
	r, _, _ := pOpenClipboard.Call(0)
	if r == 0 {
		return ""
	}
	defer pCloseClipboard.Call()

	h, _, _ := pGetClipboardData.Call(13) // CF_UNICODETEXT
	if h == 0 {
		return ""
	}

	sz, _, _ := pGlobalSize.Call(h)
	if sz < 2 {
		return ""
	}
	nWords := sz / 2

	ptr, _, _ := pGlobalLock.Call(h)
	if ptr == 0 {
		return ""
	}
	defer pGlobalUnlock.Call(h)

	// Copy into a Go-managed buffer via RtlMoveMemory.
	// ptr stays as uintptr (syscall source arg) — no uintptr→unsafe.Pointer conversion.
	buf := make([]uint16, nWords)
	pRtlMoveMemory.Call(uintptr(unsafe.Pointer(&buf[0])), ptr, sz)

	return syscall.UTF16ToString(buf)
}

func readForegroundWindowTitle() string {
	hwnd, _, _ := pGetForegroundWindow.Call()
	if hwnd == 0 {
		return ""
	}
	length, _, _ := pGetWindowTextLengthW.Call(hwnd)
	if length == 0 {
		return ""
	}
	buf := make([]uint16, length+1)
	pGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), length+1)
	return syscall.UTF16ToString(buf)
}

func readOSVersion() string {
	var info rtlOsVersionInfoW
	info.OSVersionInfoSize = uint32(unsafe.Sizeof(info))
	pRtlGetVersion.Call(uintptr(unsafe.Pointer(&info)))
	if info.BuildNumber >= 22000 {
		return "Windows 11"
	}
	return "Windows 10"
}

func readLocale() string {
	var buf [85]uint16
	pGetUserDefaultLocaleName.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf[:])
}

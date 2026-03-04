//go:build windows

package main

import (
	"syscall"
	"time"
	"unsafe"
)

const (
	cfUnicodeText  = 13
	gmemMoveable   = 0x0002
	inputKeyboard  = 1
	keyeventfKeyup = 0x0002
	vkShift        = 0x10
	vkControl      = 0x11
	vkAlt          = 0x12
	vkC            = 0x43
	vkLWin         = 0x5B
	vkRWin         = 0x5C
)

var (
	kernel32          = syscall.NewLazyDLL("kernel32.dll")
	pOpenClipboard    = user32.NewProc("OpenClipboard")
	pCloseClipboard   = user32.NewProc("CloseClipboard")
	pEmptyClipboard   = user32.NewProc("EmptyClipboard")
	pGetClipboardData = user32.NewProc("GetClipboardData")
	pSetClipboardData = user32.NewProc("SetClipboardData")
	pGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	pGlobalFree       = kernel32.NewProc("GlobalFree")
	pGlobalLock       = kernel32.NewProc("GlobalLock")
	pGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
	pSendInput        = user32.NewProc("SendInput")
	pGetAsyncKeyState = user32.NewProc("GetAsyncKeyState")
	pGetClassName     = user32.NewProc("GetClassNameW")
)

type keyboardInput struct {
	Vk        uint16
	Scan      uint16
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
	_         [8]byte // pad union to MOUSEINPUT size (28 bytes)
}

type winInput struct {
	Type uint32
	_    uint32 // align union to 8-byte boundary
	Ki   keyboardInput
}

// releaseModifiers injects key-up events for the common hotkey modifiers.
// This is necessary when the hotkey includes Alt: pressing Alt activates
// Windows menu mode in the focused window, which shifts UIA focus away from
// the text area to the menu bar. Releasing the modifiers before capture
// clears menu mode and ensures both UIA and simulated Ctrl+C work correctly.
func releaseModifiers() {
	all := [5]uint16{vkShift, vkControl, vkAlt, vkLWin, vkRWin}
	var inputs [5]winInput
	n := 0
	for _, vk := range all {
		state, _, _ := pGetAsyncKeyState.Call(uintptr(vk))
		if state&0x8000 != 0 { // high bit set = key is currently down
			inputs[n] = winInput{Type: inputKeyboard, Ki: keyboardInput{Vk: vk, Flags: keyeventfKeyup}}
			n++
		}
	}
	if n > 0 {
		pSendInput.Call(
			uintptr(n),
			uintptr(unsafe.Pointer(&inputs[0])),
			unsafe.Sizeof(inputs[0]),
		)
	}
}

func simulateCtrlC() {
	inputs := [4]winInput{
		{Type: inputKeyboard, Ki: keyboardInput{Vk: vkControl}},
		{Type: inputKeyboard, Ki: keyboardInput{Vk: vkC}},
		{Type: inputKeyboard, Ki: keyboardInput{Vk: vkC, Flags: keyeventfKeyup}},
		{Type: inputKeyboard, Ki: keyboardInput{Vk: vkControl, Flags: keyeventfKeyup}},
	}
	pSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		unsafe.Sizeof(inputs[0]),
	)
}

// readClipboardText returns the current clipboard text and whether the
// clipboard contained text at all (false for images, files, or errors).
func readClipboardText() (string, bool) {
	if r, _, _ := pOpenClipboard.Call(0); r == 0 {
		return "", false
	}
	defer pCloseClipboard.Call()
	h, _, _ := pGetClipboardData.Call(cfUnicodeText)
	if h == 0 {
		return "", false
	}
	p, _, _ := pGlobalLock.Call(h)
	if p == 0 {
		return "", false
	}
	defer pGlobalUnlock.Call(h)
	text := syscall.UTF16ToString((*[1 << 20]uint16)(unsafe.Pointer(p))[:])
	return text, true
}

func writeClipboardText(text string) {
	utf16, err := syscall.UTF16FromString(text)
	if err != nil {
		return
	}
	h, _, _ := pGlobalAlloc.Call(gmemMoveable, uintptr(len(utf16)*2))
	if h == 0 {
		return
	}
	p, _, _ := pGlobalLock.Call(h)
	if p == 0 {
		pGlobalFree.Call(h)
		return
	}
	copy((*[1 << 20]uint16)(unsafe.Pointer(p))[:len(utf16)], utf16)
	pGlobalUnlock.Call(h)
	if r, _, _ := pOpenClipboard.Call(0); r == 0 {
		pGlobalFree.Call(h)
		return
	}
	pEmptyClipboard.Call()
	pSetClipboardData.Call(cfUnicodeText, h) // clipboard takes ownership — do NOT GlobalFree
	pCloseClipboard.Call()
}

// isFocusedWindowConsole reports whether the foreground window is a traditional
// Windows console (class "ConsoleWindowClass"). This covers cmd.exe, legacy
// PowerShell, and any other app hosted by conhost.exe.
// Windows Terminal uses a different class and is handled by Tier 1 (UIA).
func isFocusedWindowConsole() bool {
	hwnd, _, _ := pGetForegroundWindow.Call()
	if hwnd == 0 {
		return false
	}
	var buf [256]uint16
	pGetClassName.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf[:]) == "ConsoleWindowClass"
}

// captureSelectedText returns the text currently selected in the focused
// window. It uses a two-tier strategy:
//
//   - Tier 1 (UIA): asks the focused window directly via Windows UI Automation.
//     Works for Chrome 126+, Edge, Windows Terminal, and most native apps.
//     No clipboard touched.
//
//   - Tier 2 (clipboard diff): simulates Ctrl+C, reads the clipboard, then
//     restores the original contents. Fallback for Firefox, older Chrome, and
//     legacy Win32 controls that don't expose a UIA TextPattern.
//     Skipped for console windows (Ctrl+C would send SIGINT instead of copying)
//     and when clipboardCapture is false.
func captureSelectedText(clipboardCapture bool) (string, bool) {
	// Tier 1: query UIA before releasing any modifiers. At hotkey-fire time
	// the target app has received Alt key-down but not key-up, so focus is
	// still on the text element. Releasing Alt first would cause the app to
	// process the key-up, post SC_KEYMENU, and shift focus to the menu bar.
	if text, ok := selectedTextViaUIA(); ok {
		return text, true
	}
	if !clipboardCapture {
		debugf("captureSelectedText: Tier 2 disabled in config")
		return "", false
	}
	if isFocusedWindowConsole() {
		debugf("captureSelectedText: skipping Ctrl+C simulation for console window")
		return "", false
	}
	// Tier 2: release held modifiers immediately before simulating Ctrl+C,
	// with no sleep. A sleep here would give the target app's message loop
	// time to process the Alt key-up, activate the menu bar, and steal focus
	// from the text area — causing Ctrl+C to land on the menu instead.
	releaseModifiers()
	return captureViaClipboard()
}

// captureViaClipboard is the Tier 2 capture path: save clipboard, simulate
// Ctrl+C, read new clipboard, restore original.
// Non-text clipboard contents (images, files) are not restored — known limitation.
func captureViaClipboard() (string, bool) {
	old, hadText := readClipboardText()
	defer func() {
		if hadText {
			writeClipboardText(old)
		}
	}()
	debugf("captureViaClipboard: simulating Ctrl+C")
	simulateCtrlC()
	time.Sleep(150 * time.Millisecond)
	newText, _ := readClipboardText()
	debugf("captureViaClipboard: read %d bytes, hasContext=%v", len(newText), newText != "" && newText != old)
	if newText != "" && newText != old {
		return newText, true
	}
	return "", false
}

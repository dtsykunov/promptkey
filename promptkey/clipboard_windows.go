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
	vkControl      = 0x11
	vkC            = 0x43
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

// captureSelectedText saves the clipboard, simulates Ctrl+C, reads the new
// clipboard, restores the original, and returns the selected text if captured.
// Non-text clipboard contents (images, files) are not restored — known limitation.
func captureSelectedText() (string, bool) {
	old, hadText := readClipboardText()
	defer func() {
		if hadText {
			writeClipboardText(old)
		}
	}()
	debugf("captureSelectedText: simulating Ctrl+C")
	simulateCtrlC()
	time.Sleep(150 * time.Millisecond)
	newText, _ := readClipboardText()
	debugf("captureSelectedText: read %d bytes, hasContext=%v", len(newText), newText != "" && newText != old)
	if newText != "" && newText != old {
		return newText, true
	}
	return "", false
}

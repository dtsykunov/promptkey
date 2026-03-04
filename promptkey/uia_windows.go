//go:build windows

package main

// selectedTextViaUIA captures the currently selected text in the focused
// window using the Windows UI Automation (UIA) API. This is Tier 1 of text
// capture and works for Chrome 126+, Edge, and most native Windows apps.
//
// UIA is the accessibility API used by screen readers (Narrator, NVDA). Unlike
// the Ctrl+C clipboard approach, it asks the app directly for the selected
// text, so browser security restrictions on synthetic input don't apply.
//
// If the focused element doesn't implement the UIA TextPattern (older Chrome,
// Firefox, some apps), this returns ("", false) and the caller falls through to
// the clipboard-based Tier 2 approach.
//
// All five COM interfaces needed here are declared with explicit vtable structs.
// Vtable field order must exactly match UIAutomationClient.h from the Windows SDK.
// Unused methods before the one we call are grouped into anonymous padding fields.

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
)

// GUIDs from UIAutomationClient.h and UIAutomationCoreApi.h.
var (
	clsidCUIAutomation = &ole.GUID{
		Data1: 0xFF48DBA4,
		Data2: 0x60EF,
		Data3: 0x4201,
		Data4: [8]byte{0xAA, 0x87, 0x54, 0x10, 0x3E, 0xEF, 0x59, 0x4E},
	}
	iidIUIAutomation = &ole.GUID{
		Data1: 0x30CBE57D,
		Data2: 0xD9D0,
		Data3: 0x452A,
		Data4: [8]byte{0xAB, 0x13, 0x7A, 0xC5, 0xAC, 0x48, 0x25, 0xEE},
	}
	iidIUIAutomationTextPattern = &ole.GUID{
		Data1: 0x32EBA289,
		Data2: 0x3583,
		Data3: 0x42C9,
		Data4: [8]byte{0x9C, 0x59, 0x3B, 0x6C, 0xB1, 0x74, 0x0B, 0x33},
	}
)

// UIA_TextPatternId identifies the TextPattern control pattern (from UIAutomationCoreApi.h).
const uiaTextPatternID uintptr = 10014

var (
	oleaut32       = syscall.NewLazyDLL("oleaut32.dll")
	pSysFreeString = oleaut32.NewProc("SysFreeString")
)

// --- COM interface vtable structs ---
//
// Each COM interface is a struct whose first field is a pointer to a vtable.
// The vtable is a contiguous array of function pointers in the order defined
// by the interface's IDL. We only declare fields up to the last method we call.

type iUIAutomationVtbl struct {
	_ [8]uintptr // [0] QueryInterface  [1] AddRef  [2] Release
	//              [3] CompareElements  [4] CompareRuntimeIds
	//              [5] GetRootElement   [6] ElementFromHandle  [7] ElementFromPoint
	GetFocusedElement uintptr // [8]
}

type iUIAutomationElementVtbl struct {
	_ [14]uintptr // [0] QueryInterface  [1] AddRef  [2] Release
	//               [3] SetFocus  [4] GetRuntimeId
	//               [5] FindFirst  [6] FindAll
	//               [7] FindFirstBuildCache  [8] FindAllBuildCache  [9] BuildUpdatedCache
	//               [10] GetCurrentPropertyValue  [11] GetCurrentPropertyValueEx
	//               [12] GetCachedPropertyValue   [13] GetCachedPropertyValueEx
	GetCurrentPatternAs uintptr // [14]
}

type iUIAutomationTextPatternVtbl struct {
	_            [3]uintptr // [0] QueryInterface  [1] AddRef  [2] Release
	GetSelection uintptr    // [3]
}

type iUIAutomationTextRangeArrayVtbl struct {
	_          [3]uintptr // [0] QueryInterface  [1] AddRef  [2] Release
	get_Length uintptr    // [3]
	GetElement uintptr    // [4]
}

type iUIAutomationTextRangeVtbl struct {
	_ [12]uintptr // [0] QueryInterface  [1] AddRef  [2] Release
	//               [3] Clone  [4] Compare  [5] CompareEndpoints
	//               [6] ExpandToEnclosingUnit  [7] FindAttribute  [8] FindText
	//               [9] GetAttributeValue  [10] GetBoundingRectangles  [11] GetEnclosingElement
	GetText uintptr // [12]
}

// --- COM interface wrappers ---

type iUIAutomation struct{ vtbl *iUIAutomationVtbl }
type iUIAutomationElement struct{ vtbl *iUIAutomationElementVtbl }
type iUIAutomationTextPattern struct{ vtbl *iUIAutomationTextPatternVtbl }
type iUIAutomationTextRangeArray struct {
	vtbl *iUIAutomationTextRangeArrayVtbl
}
type iUIAutomationTextRange struct{ vtbl *iUIAutomationTextRangeVtbl }

// release calls IUnknown::Release on any of our COM wrappers.
// All COM interfaces inherit from IUnknown, so Release is always at vtable index [2].
// We cast to *ole.IUnknown which calls through index [2] via go-ole.
func comRelease(p unsafe.Pointer) {
	(*ole.IUnknown)(p).Release()
}

func (p *iUIAutomation) release() { comRelease(unsafe.Pointer(p)) }

func (p *iUIAutomation) getFocusedElement() (*iUIAutomationElement, error) {
	var out *iUIAutomationElement
	hr, _, _ := syscall.SyscallN(
		p.vtbl.GetFocusedElement,
		uintptr(unsafe.Pointer(p)),
		uintptr(unsafe.Pointer(&out)),
	)
	if hr != 0 {
		return nil, fmt.Errorf("GetFocusedElement: 0x%08X", hr)
	}
	return out, nil
}

func (p *iUIAutomationElement) release() { comRelease(unsafe.Pointer(p)) }

func (p *iUIAutomationElement) getCurrentPatternAs(patternID uintptr, iid *ole.GUID) (unsafe.Pointer, error) {
	var out unsafe.Pointer
	hr, _, _ := syscall.SyscallN(
		p.vtbl.GetCurrentPatternAs,
		uintptr(unsafe.Pointer(p)),
		patternID,
		uintptr(unsafe.Pointer(iid)),
		uintptr(unsafe.Pointer(&out)),
	)
	if hr != 0 {
		return nil, fmt.Errorf("GetCurrentPatternAs: 0x%08X", hr)
	}
	return out, nil
}

func (p *iUIAutomationTextPattern) release() { comRelease(unsafe.Pointer(p)) }

func (p *iUIAutomationTextPattern) getSelection() (*iUIAutomationTextRangeArray, error) {
	var out *iUIAutomationTextRangeArray
	hr, _, _ := syscall.SyscallN(
		p.vtbl.GetSelection,
		uintptr(unsafe.Pointer(p)),
		uintptr(unsafe.Pointer(&out)),
	)
	if hr != 0 {
		return nil, fmt.Errorf("GetSelection: 0x%08X", hr)
	}
	return out, nil
}

func (p *iUIAutomationTextRangeArray) release() { comRelease(unsafe.Pointer(p)) }

func (p *iUIAutomationTextRangeArray) length() (int, error) {
	var n int32
	hr, _, _ := syscall.SyscallN(
		p.vtbl.get_Length,
		uintptr(unsafe.Pointer(p)),
		uintptr(unsafe.Pointer(&n)),
	)
	if hr != 0 {
		return 0, fmt.Errorf("get_Length: 0x%08X", hr)
	}
	return int(n), nil
}

func (p *iUIAutomationTextRangeArray) getElement(i int) (*iUIAutomationTextRange, error) {
	var out *iUIAutomationTextRange
	hr, _, _ := syscall.SyscallN(
		p.vtbl.GetElement,
		uintptr(unsafe.Pointer(p)),
		uintptr(i),
		uintptr(unsafe.Pointer(&out)),
	)
	if hr != 0 {
		return nil, fmt.Errorf("GetElement(%d): 0x%08X", i, hr)
	}
	return out, nil
}

func (p *iUIAutomationTextRange) release() { comRelease(unsafe.Pointer(p)) }

func (p *iUIAutomationTextRange) getText(maxLen int) (string, error) {
	var bstr *uint16
	hr, _, _ := syscall.SyscallN(
		p.vtbl.GetText,
		uintptr(unsafe.Pointer(p)),
		uintptr(maxLen),
		uintptr(unsafe.Pointer(&bstr)),
	)
	if hr != 0 {
		return "", fmt.Errorf("GetText: 0x%08X", hr)
	}
	if bstr == nil {
		return "", nil
	}
	defer pSysFreeString.Call(uintptr(unsafe.Pointer(bstr)))
	return syscall.UTF16ToString((*[1 << 20]uint16)(unsafe.Pointer(bstr))[:]), nil
}

// selectedTextViaUIA queries the focused window for its current text selection
// via the Windows UI Automation API. It is the Tier 1 capture path.
func selectedTextViaUIA() (string, bool) {
	// The hotkey goroutine already called LockOSThread; this nested call is
	// safe — LockOSThread stacks, and the matching UnlockOSThread just
	// decrements the count, keeping the goroutine pinned for the message loop.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// CoInitializeEx returns S_FALSE (0x1) if COM is already initialized on
	// this thread; per MSDN, CoUninitialize must be called in both the S_OK
	// and S_FALSE cases.
	err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED)
	if err != nil {
		oleErr, ok := err.(*ole.OleError)
		if !ok || oleErr.Code() != 0x00000001 { // 0x1 = S_FALSE
			debugf("uia: CoInitializeEx: %v", err)
			return "", false
		}
	}
	defer ole.CoUninitialize()

	unknown, err := ole.CreateInstance(clsidCUIAutomation, iidIUIAutomation)
	if err != nil {
		debugf("uia: CoCreateInstance: %v", err)
		return "", false
	}
	uia := (*iUIAutomation)(unsafe.Pointer(unknown))
	defer uia.release()

	elem, err := uia.getFocusedElement()
	if err != nil {
		debugf("uia: %v", err)
		return "", false
	}
	defer elem.release()

	raw, err := elem.getCurrentPatternAs(uiaTextPatternID, iidIUIAutomationTextPattern)
	if err != nil || raw == nil {
		debugf("uia: TextPattern not supported by focused element")
		return "", false
	}
	pattern := (*iUIAutomationTextPattern)(raw)
	defer pattern.release()

	ranges, err := pattern.getSelection()
	if err != nil || ranges == nil {
		debugf("uia: getSelection: %v", err)
		return "", false
	}
	defer ranges.release()

	n, err := ranges.length()
	if err != nil || n == 0 {
		return "", false
	}

	r0, err := ranges.getElement(0)
	if err != nil {
		debugf("uia: getElement: %v", err)
		return "", false
	}
	defer r0.release()

	text, err := r0.getText(-1)
	if err != nil {
		debugf("uia: getText: %v", err)
		return "", false
	}
	if text == "" {
		return "", false
	}
	debugf("uia: captured %d bytes via TextPattern", len(text))
	return text, true
}

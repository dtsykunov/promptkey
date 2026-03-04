package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const popupW, popupH = 480, 56

type App struct {
	ctx          context.Context
	cfg          Config
	selectedText string
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.cfg = loadConfig()
	a.saveConfig() // ensure config file exists on first run so the user can find and edit it
	debugf("app starting")
	a.setupTray()
	a.startHotkey(a.showPopup)
}

func (a *App) showPopup() {
	x, y := getCursorPos()
	debugf("showPopup: cursor pos (%d, %d)", x, y)
	var text string
	var hasContext bool
	if a.cfg.CaptureContext {
		text, hasContext = captureSelectedText(a.cfg.ClipboardCapture)
		debugf("showPopup: hasContext=%v", hasContext)
		debugf("showPopup: captured: %q", text)
	}
	a.selectedText = text
	px, py := a.popupPosition(x, y, popupW, popupH)
	debugf("showPopup: popup position (%d, %d)", px, py)
	runtime.WindowSetSize(a.ctx, popupW, popupH)
	runtime.WindowSetPosition(a.ctx, px, py)
	runtime.WindowShow(a.ctx)
	a.startFocusWatcher()
	runtime.EventsEmit(a.ctx, "popup:open", hasContext)
}

// SendPrompt is called by the frontend on submit.
// Step 4: echoes to log. Step 5 will replace with streaming AI call.
func (a *App) SendPrompt(instructions string) {
	debugf("SendPrompt: instructions=%q, hasContext=%v", instructions, a.selectedText != "")
	msg := instructions
	if a.selectedText != "" {
		msg += "\n\n[context] " + a.selectedText
	}
	runtime.LogPrint(a.ctx, msg)
	a.selectedText = ""
}

func (a *App) hidePopup() {
	runtime.WindowHide(a.ctx)
}

func (a *App) popupPosition(cx, cy, w, h int) (int, int) {
	x, y := cx+16, cy

	screens, err := runtime.ScreenGetAll(a.ctx)
	if err != nil || len(screens) == 0 {
		return x, y
	}

	// Use the primary screen (or first available) to clamp the popup within bounds.
	screen := screens[0]
	for _, s := range screens {
		if s.IsPrimary {
			screen = s
			break
		}
	}

	sw, sh := screen.Size.Width, screen.Size.Height
	if x+w > sw-8 {
		x = sw - 8 - w
	}
	if y+h > sh-8 {
		y = sh - 8 - h
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return x, y
}

package main

import (
	"context"
	"errors"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const popupW, popupH = 480, 56

type App struct {
	ctx          context.Context
	cfg          Config
	cancelStream context.CancelFunc
	lastPrompt   string
	cursorX      int
	cursorY      int
	inResultView bool
	stopFocus    chan struct{}
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.cfg = loadConfig()
	a.saveConfig()
	debugf("app starting")
	a.setupTray()
	a.startHotkey(a.showPopup)
}

func (a *App) showPopup() {
	x, y := getCursorPos()
	debugf("showPopup: cursor pos (%d, %d)", x, y)
	a.cursorX, a.cursorY = x, y
	a.inResultView = false
	px, py := a.calcPosition(x, y, popupW, popupH)
	debugf("showPopup: popup position (%d, %d)", px, py)

	// Stop any previous focus watcher.
	if a.stopFocus != nil {
		close(a.stopFocus)
	}
	a.stopFocus = make(chan struct{})

	runtime.WindowSetSize(a.ctx, popupW, popupH)
	runtime.WindowSetPosition(a.ctx, px, py)
	runtime.WindowShow(a.ctx)
	a.startFocusWatcher(a.stopFocus)
	runtime.EventsEmit(a.ctx, "popup:open")
}

// SendPrompt is called by the frontend on submit.
func (a *App) SendPrompt(instructions string) {
	debugf("SendPrompt: instructions=%q", instructions)

	// Stop focus watcher — result window doesn't close on focus loss.
	if a.stopFocus != nil {
		close(a.stopFocus)
		a.stopFocus = nil
	}

	// Cancel any in-flight stream.
	if a.cancelStream != nil {
		a.cancelStream()
	}
	a.lastPrompt = instructions

	// Resolve result window size (use config or defaults).
	rw, rh := a.cfg.ResultW, a.cfg.ResultH
	if rw <= 0 {
		rw = defaultResultW
	}
	if rh <= 0 {
		rh = defaultResultH
	}

	// Only reposition on the initial popup→result transition, not on retry.
	if !a.inResultView {
		rx, ry := a.calcPosition(a.cursorX-rw, a.cursorY-rh, rw, rh)
		runtime.WindowSetSize(a.ctx, rw, rh)
		runtime.WindowSetPosition(a.ctx, rx, ry)
		a.inResultView = true
	}
	runtime.EventsEmit(a.ctx, "result:open")

	p, err := a.activeProvider()
	if err != nil {
		runtime.EventsEmit(a.ctx, "ai:error", err.Error())
		return
	}

	ctx, cancel := context.WithCancel(a.ctx)
	a.cancelStream = cancel
	go func() {
		defer cancel()
		for ev := range streamCompletion(ctx, p, instructions) {
			if ev.err != nil {
				if !errors.Is(ev.err, context.Canceled) {
					runtime.EventsEmit(a.ctx, "ai:error", ev.err.Error())
				}
				return
			}
			runtime.EventsEmit(a.ctx, "ai:chunk", ev.chunk)
		}
		runtime.EventsEmit(a.ctx, "ai:done")
	}()
}

// Retry re-sends the last prompt.
func (a *App) Retry() {
	if a.lastPrompt != "" {
		a.SendPrompt(a.lastPrompt)
	}
}

// SaveResultSize persists the user-resized result window dimensions.
func (a *App) SaveResultSize(w, h int) {
	if w > 0 && h > 0 {
		a.cfg.ResultW = w
		a.cfg.ResultH = h
		a.saveConfig()
	}
}

func (a *App) hidePopup() {
	runtime.WindowHide(a.ctx)
}

// calcPosition returns the top-left corner for a window of size (w, h)
// anchored at (ax, ay), clamped within the primary screen.
func (a *App) calcPosition(ax, ay, w, h int) (int, int) {
	x, y := ax, ay

	screens, err := runtime.ScreenGetAll(a.ctx)
	if err != nil || len(screens) == 0 {
		return x, y
	}

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

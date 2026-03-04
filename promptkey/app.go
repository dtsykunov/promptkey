package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const popupW, popupH = 480, 56
const popupHCtx = 76 // popupH + 20px context bar
const settingsW, settingsH = 560, 480

// PopupBar carries context bar data emitted with popup:open.
type PopupBar struct {
	HasClipboard bool   `json:"hasClipboard"`
	App          string `json:"app"`
	DateTime     string `json:"datetime"`
}

type App struct {
	ctx            context.Context
	cfg            Config
	cancelStream   context.CancelFunc
	lastPrompt     string
	lastCtx        Context
	cursorX        int
	cursorY        int
	inResultView   bool
	stopFocus      chan struct{}
	hotkeyThreadID uint32 // atomic; written by hotkey goroutine
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.cfg = loadConfig()
	if !a.cfg.Context.Initialized {
		a.cfg.Context = defaultContextConfig()
	}
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

	// Capture context on hotkey press.
	a.lastCtx = CaptureContext(a.cfg.Context)
	bar := PopupBar{
		HasClipboard: a.cfg.Context.Enabled && a.cfg.Context.Clipboard && a.lastCtx.Clipboard != "",
		App:          a.lastCtx.App,
		DateTime:     a.lastCtx.DateTime,
	}
	hasBar := bar.HasClipboard || bar.App != "" || bar.DateTime != ""
	h := popupH
	if hasBar {
		h = popupHCtx
	}

	px, py := a.calcPosition(x, y, popupW, h)
	debugf("showPopup: popup position (%d, %d)", px, py)

	// Stop any previous focus watcher.
	if a.stopFocus != nil {
		close(a.stopFocus)
	}
	a.stopFocus = make(chan struct{})

	runtime.WindowSetAlwaysOnTop(a.ctx, true)
	runtime.WindowSetSize(a.ctx, popupW, h)
	runtime.WindowSetPosition(a.ctx, px, py)
	runtime.WindowShow(a.ctx)
	a.startFocusWatcher(a.stopFocus)
	runtime.EventsEmit(a.ctx, "popup:open", bar)
}

func (a *App) showSettings() {
	runtime.WindowSetAlwaysOnTop(a.ctx, false)
	runtime.WindowSetSize(a.ctx, settingsW, settingsH)
	runtime.WindowCenter(a.ctx)
	runtime.WindowShow(a.ctx)
	runtime.EventsEmit(a.ctx, "settings:open")
}

// GetConfig returns the current config to the frontend.
func (a *App) GetConfig() Config { return a.cfg }

// SaveSettings validates and persists a new config, restarting the hotkey if it changed.
func (a *App) SaveSettings(cfg Config) string {
	key := cfg.Hotkey
	if key == "" {
		key = defaultHotkey
	}
	if _, _, err := parseHotkey(key); err != nil {
		return fmt.Sprintf("invalid hotkey: %v", err)
	}
	hotkeyChanged := cfg.Hotkey != a.cfg.Hotkey
	a.cfg = cfg
	a.saveConfig()
	if hotkeyChanged {
		a.resetHotkey()
	}
	return ""
}

// modelsResponse is the OpenAI /models response shape.
type modelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// FetchModels fetches available model IDs from an OpenAI-compatible /models endpoint.
func (a *App) FetchModels(baseURL, apiKey string) ([]string, error) {
	req, err := http.NewRequest(http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api error %d: %s", resp.StatusCode, body)
	}
	var mr modelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&mr); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	ids := make([]string, 0, len(mr.Data))
	for _, d := range mr.Data {
		ids = append(ids, d.ID)
	}
	return ids, nil
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
	p.SystemPrompt = RenderTemplate(p.SystemPrompt, a.lastCtx)

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

//go:build windows

package main

import (
	_ "embed"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed build/windows/icon.ico
var trayIcon []byte

func (a *App) setupTray() {
	go systray.Run(a.onTrayReady, nil)
}

func (a *App) onTrayReady() {
	systray.SetIcon(trayIcon)
	systray.SetTooltip("PromptKey")

	mCapture := systray.AddMenuItemCheckbox("Capture context", "Capture selected text when the popup opens", a.cfg.CaptureContext)
	mClipboard := systray.AddMenuItemCheckbox("Clipboard capture", "Fall back to Ctrl+C simulation when UIA fails (disabled automatically for console windows)", a.cfg.ClipboardCapture)
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit PromptKey")

	go func() {
		for {
			select {
			case <-mCapture.ClickedCh:
				if mCapture.Checked() {
					mCapture.Uncheck()
					a.cfg.CaptureContext = false
				} else {
					mCapture.Check()
					a.cfg.CaptureContext = true
				}
				a.saveConfig()
			case <-mClipboard.ClickedCh:
				if mClipboard.Checked() {
					mClipboard.Uncheck()
					a.cfg.ClipboardCapture = false
				} else {
					mClipboard.Check()
					a.cfg.ClipboardCapture = true
				}
				a.saveConfig()
			case <-mQuit.ClickedCh:
				systray.Quit()
				runtime.Quit(a.ctx)
				return
			}
		}
	}()
}

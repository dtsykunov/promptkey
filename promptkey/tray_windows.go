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

	mQuit := systray.AddMenuItem("Quit", "Quit PromptKey")

	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
		runtime.Quit(a.ctx)
	}()
}

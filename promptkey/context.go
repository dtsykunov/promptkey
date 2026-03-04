package main

import (
	"strings"
	"time"
)

// Context holds captured context variables for template rendering.
type Context struct {
	Clipboard string
	App       string
	Date      string
	Time      string
	DateTime  string
	OS        string
	Locale    string
}

// CaptureContext fills date/time fields from time.Now() and delegates
// platform-specific fields (clipboard, app, OS, locale) to platformCaptureContext.
func CaptureContext(cfg ContextConfig) Context {
	if !cfg.Enabled {
		return Context{}
	}
	now := time.Now()
	ctx := platformCaptureContext(cfg)
	if cfg.DateTime {
		ctx.Date = now.Format("2006-01-02")
		ctx.Time = now.Format("15:04")
		ctx.DateTime = now.Format("2006-01-02 15:04")
	}
	return ctx
}

// RenderTemplate replaces {{variable}} placeholders in s with values from ctx.
func RenderTemplate(s string, ctx Context) string {
	r := strings.NewReplacer(
		"{{clipboard}}", ctx.Clipboard,
		"{{app}}", ctx.App,
		"{{date}}", ctx.Date,
		"{{time}}", ctx.Time,
		"{{datetime}}", ctx.DateTime,
		"{{os}}", ctx.OS,
		"{{locale}}", ctx.Locale,
	)
	return r.Replace(s)
}

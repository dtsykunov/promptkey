# PromptKey

Press a hotkey. A floating prompt appears near your cursor. Type instructions. Hit Enter. Get an AI response in a result overlay with Copy and Close buttons.

Lives in the system tray. Works on macOS and Windows. Supports any OpenAI-compatible endpoint — cloud or self-hosted.

---

## How It Works

1. Press the hotkey (default: `Ctrl+Alt+\``)
2. A floating input appears near your cursor
3. Type instructions and press Enter or click ↵
4. The AI response streams into a result overlay
5. **Copy** the result to clipboard, or **Close** to dismiss

---

## Architecture

**Three windows:** a frameless always-on-top popup (shown on hotkey, near cursor), a frameless always-on-top result window (replaces popup after submit), and a normal settings window (opened from tray). Never show popup and result at the same time.

**Hotkey flow:** Go detects hotkey → gets mouse position → shows popup at calculated position → emits Wails event `popup:open` → frontend auto-focuses the input.

**AI flow:** `app.SendPrompt(instructions string)` (bound Go method) → goroutine streams response → `runtime.EventsEmit(ctx, "ai:chunk", chunk)` per token → `ai:done` on completion → frontend transitions to result view.

---

## Prompt Construction

- **System prompt:** configurable per provider, defaults to `"You are a concise, helpful assistant."`
- **User message:** the instructions typed by the user

---

## Config

Stored at:
- macOS: `~/Library/Application Support/promptkey/config.json`
- Windows: `%APPDATA%\promptkey\config.json`

```json
{
  "hotkey": "ctrl+alt+`",
  "theme": "auto",
  "providers": [
    {
      "name": "Claude",
      "baseURL": "https://api.anthropic.com/v1",
      "apiKey": "sk-...",
      "model": "claude-sonnet-4-6",
      "systemPrompt": "You are a concise, helpful assistant."
    },
    {
      "name": "Ollama",
      "baseURL": "http://localhost:11434/v1",
      "apiKey": "",
      "model": "llama3",
      "systemPrompt": ""
    }
  ],
  "activeProvider": "Claude"
}
```

Any OpenAI-compatible provider works: OpenAI, Anthropic, Mistral, Groq, Ollama, LM Studio, Jan, and others.

---

## Platform Notes

**macOS:** Set `LSUIElement = true` in `Info.plist` (no Dock icon). Accessibility permission required for the global hotkey — prompt the user on first launch with a direct link to System Settings → Privacy & Security → Accessibility.

**Windows:** No taskbar entry in tray mode. Tray icon must be `.ico`. Use `RegisterHotKey` via `golang.org/x/sys/windows`.

---

## Build Order

1. Tray icon appears, Quit exits cleanly
2. Hotkey shows/hides a frameless window near cursor
3. Text input in popup — Escape closes, Enter submits (echo only, no AI yet)
4. Real streaming AI call, result shown in result view with Copy + Close buttons
5. Config system — load/save JSON, settings window to manage providers and hotkey
6. Polish — error states, permission prompts, transitions, opacity

---

## Building from Source

**Prerequisites:** Go 1.21+, Node.js 18+, [Wails CLI](https://wails.io/docs/gettingstarted/installation)

```bash
wails dev                                # development with hot reload
wails build                              # production binary, current platform
wails build -platform darwin/universal   # macOS universal (Intel + Apple Silicon)
wails build -platform windows/amd64
```

---

## Permissions

**macOS only:** Accessibility access is required (System Settings → Privacy & Security → Accessibility) for the global hotkey. The app will prompt on first launch.

---

MIT License

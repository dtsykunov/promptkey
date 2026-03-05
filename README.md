# PromptKey

> **This project is on hiatus and not actively maintained.**

Press a hotkey. A floating prompt appears near your cursor. Type instructions. Hit Enter. Get an AI response in a result overlay with Copy and Close buttons.

Lives in the system tray. **Windows only.** Supports any OpenAI-compatible endpoint — cloud or self-hosted.

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

**Context capture:** On each hotkey press, clipboard content, active window title, date/time, OS version, and locale are captured and injected into the system prompt via `{{variable}}` placeholders.

---

## Config

Stored at `%APPDATA%\promptkey\config.json`.

```json
{
  "hotkey": "ctrl+alt+`",
  "providers": [
    {
      "name": "Claude",
      "baseURL": "https://api.anthropic.com/v1",
      "apiKey": "sk-...",
      "model": "claude-sonnet-4-6",
      "systemPrompt": "You are a quick-reference assistant..."
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

Any OpenAI-compatible provider works: OpenAI, Anthropic (via compatible endpoint), Mistral, Groq, Perplexity, Ollama, LM Studio, and others.

Template variables available in system prompts: `{{clipboard}}` `{{app}}` `{{date}}` `{{time}}` `{{datetime}}` `{{os}}` `{{locale}}`

---

## Building from Source

**Prerequisites:** Go 1.21+, Node.js 18+, [Wails CLI](https://wails.io/docs/gettingstarted/installation)

```bash
# from promptkey/
make build   # release binary (no console window)
make debug   # debug binary (console window + verbose logs)
```

---

MIT License

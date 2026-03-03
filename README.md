# PromptKey

Press a hotkey. A floating prompt appears near your cursor. Optionally type instructions. Hit Enter. Get an AI response in a result overlay with Copy and Close buttons.

If you have text selected anywhere on screen when you trigger the hotkey, it is automatically captured and sent as context alongside your instructions.

Lives in the system tray. Works on macOS and Windows. Supports any OpenAI-compatible endpoint — cloud or self-hosted.

---

## How It Works

1. Select some text (optional) in any application
2. Press the hotkey (default: `Ctrl+Shift+Space`)
3. A floating input appears near your cursor — a paperclip icon indicates if context was captured
4. Type instructions, or press Enter or click ↵ to act on the selected text alone
5. The AI response streams into a result overlay
6. **Copy** the result to clipboard, or **Close** to dismiss

---

## Architecture

**Three windows:** a frameless always-on-top popup (shown on hotkey, near cursor), a frameless always-on-top result window (replaces popup after submit), and a normal settings window (opened from tray). Never show popup and result at the same time.

**Hotkey flow:** Go detects hotkey → captures selected text → gets mouse position → shows popup at calculated position → emits Wails event `popup:open` with `{ hasContext: bool }` → frontend shows paperclip indicator if context was captured and auto-focuses the input.

**AI flow:** `app.SendPrompt(instructions, selectedText string)` (bound Go method) → goroutine streams response → `runtime.EventsEmit(ctx, "ai:chunk", chunk)` per token → `ai:done` on completion → frontend transitions to result view.

---

## Selected Text Capture

Before showing the popup, Go:
1. Saves the current clipboard contents
2. Simulates Cmd+C / Ctrl+C with a short sleep for the OS to process
3. Reads the new clipboard value
4. Restores the original clipboard — always, via `defer`, even on error
5. Diffs old vs new

**Safety rules:**
- Only send text to the AI if the clipboard *changed* — the pre-existing clipboard value is never sent under any circumstance
- If no change is detected, selected text is treated as empty
- Non-text clipboard data (images, files) is not restored — document as a known limitation

**Known edge case:** if the selected text is byte-for-byte identical to what was already in the clipboard, the diff fails silently. Silent omission is preferable to leaking private clipboard data.

---

## Prompt Construction

- **System prompt:** configurable per provider, defaults to `"You are a concise, helpful assistant."`
- **User message:**
  - Instructions + selected text: `"{instructions}\n\n{selected_text}"`
  - Selected text only: `"{selected_text}"`
  - Instructions only: `"{instructions}"`

---

## Config

Stored at:
- macOS: `~/Library/Application Support/promptkey/config.json`
- Windows: `%APPDATA%\promptkey\config.json`

```json
{
  "hotkey": "ctrl+shift+space",
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

**macOS:** Set `LSUIElement = true` in `Info.plist` (no Dock icon). Accessibility permission required for the global hotkey and clipboard capture — prompt the user on first launch with a direct link to System Settings → Privacy & Security → Accessibility.

**Windows:** No taskbar entry in tray mode. Tray icon must be `.ico`. Use `RegisterHotKey` via `golang.org/x/sys/windows`.

---

## Build Order

1. Tray icon appears, Quit exits cleanly
2. Hotkey shows/hides a frameless window near cursor
3. Text input in popup — Escape closes, Enter submits (echo only, no AI yet)
4. Selected text capture — save clipboard, simulate copy, diff, restore; show paperclip indicator in popup if context was captured
5. Real streaming AI call, result shown in result view with Copy + Close buttons
6. Config system — load/save JSON, settings window to manage providers and hotkey
7. Polish — error states, permission prompts, transitions, opacity

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

**macOS only:** Accessibility access is required (System Settings → Privacy & Security → Accessibility) for the global hotkey and clipboard-based text capture. The app will prompt on first launch.

---

MIT License

# PromptKey — Full Specification

This document is the authoritative implementation reference. Where it conflicts with README.md, this spec takes precedence.

---

## Philosophy

PromptKey is a speed tool. The interaction is: select text, press hotkey, type a short question, get a short answer, copy it, done. Everything in the UX should serve that loop. Optimize for the case where the whole interaction takes under 15 seconds.

---

## Platform

**Target: Windows first.** macOS support is a second phase. All platform-specific code must be isolated behind interfaces so macOS can be wired in later without restructuring.

**Dev environment:** WSL + Nix flake. All tooling declared in `flake.nix`.

---

## Architecture

### Windows

**Three Wails windows:**

1. **Popup window** — frameless, always-on-top. Appears near cursor on hotkey. Contains only the instruction input field. Closes when user blurs, presses Escape, submits (Enter), or a new hotkey trigger occurs while a result is open.

2. **Result window** — frameless, always-on-top. Opens immediately when user submits. Lives independently of the popup (popup is hidden/closed when result opens). Stays open until user explicitly closes it (Escape, Close button, or Copy button). Does not close on blur.

3. **Settings window** — standard framed window. Opens from tray. One at a time.

**Never show the popup and result window at the same time.** When the user triggers the hotkey while a result window is open, close the result window first, then open the popup.

### Go / Wails structure

```
app.go          — App struct, all bound methods
hotkey.go       — platform hotkey registration
clipboard.go    — text capture logic
ai.go           — streaming AI client
config.go       — load/save config
tray.go         — tray setup and menu
```

No global mutable state outside the `App` struct.

---

## Hotkey Flow

1. User presses hotkey
2. If a result window is currently visible: close it
3. Capture selected text (see below)
4. Get cursor position
5. Calculate popup position (see Positioning)
6. Show popup window at calculated position
7. Emit Wails event `popup:open` with payload `{ hasContext: bool }` to frontend
8. Frontend auto-focuses the input field

### Hotkey conflict

If hotkey registration fails (already claimed by another app):
- App still starts and shows tray icon
- Tray icon changes to a **warning variant** (e.g. key icon with an exclamation badge)
- Hovering the tray icon shows tooltip: `"Hotkey unavailable: [hotkey] is in use by another app. Open Settings to change it."`
- No further notification

---

## Selected Text Capture

1. Save current clipboard contents
2. Simulate Ctrl+C (Windows) / Cmd+C (macOS)
3. Sleep 150 ms
4. Read new clipboard value
5. Restore original clipboard — always, via `defer`
6. Compare old and new clipboard contents

**Rules:**
- If clipboard changed: `selectedText = newValue`, `hasContext = true`
- If clipboard did not change: `selectedText = ""`, `hasContext = false`
- If clipboard is non-text (image, file): restore is skipped (known limitation), `hasContext = false`
- Pre-existing clipboard value is **never** sent to the AI under any circumstances
- No client-side truncation — send whatever was captured; let the API return an error if it's too large

---

## Popup Window

**Appearance:**
- Frameless, always-on-top
- Fixed width: 480 px
- Auto-height based on content (single input line + optional context badge row)
- Rounded corners, drop shadow
- Respects OS dark/light mode (can be overridden in settings)

**Contents:**
```
┌─────────────────────────────────────┐
│  [🔗]  Ask anything...              │
└─────────────────────────────────────┘
```
- Single-line text input, placeholder "Ask anything..."
- If `hasContext = true`: small paperclip icon (🔗 or SVG equivalent) appears at the left edge of the input field
- If `hasContext = false`: no icon, no other change
- No submit button, no context text preview

**Interaction:**
- Auto-focused on open
- **Enter** → submit (calls `app.SendPrompt`)
- **Escape** → close popup, discard
- **Click outside (blur)** → close popup, discard
- If input is empty and user presses Enter: no-op (do not submit empty prompt)

**Positioning:** See Positioning section.

---

## Positioning

### Popup
- Default position: 16 px to the right and 0 px below cursor
- Detect the screen containing the cursor; use that screen's bounds
- If the window would overflow right edge: shift left so right edge aligns with screen right edge minus 8 px
- If the window would overflow bottom edge: shift up so bottom edge aligns with screen bottom edge minus 8 px
- Trust Wails for multi-monitor screen detection

### Result window
- Record the **center point** of the popup window when it was shown
- Open result window centered on that same center point
- Apply the same overflow/edge correction as above

---

## Submitting a Prompt

When user presses Enter:
1. Read input text
2. Close/hide popup window
3. Call `app.SendPrompt(instructions string, selectedText string)` (bound Go method)
4. Open result window immediately at calculated position with spinner visible
5. `SendPrompt` launches a goroutine that streams the response
6. Each chunk: `runtime.EventsEmit(ctx, "ai:chunk", chunk)`
7. On completion: `runtime.EventsEmit(ctx, "ai:done", nil)`
8. On error: `runtime.EventsEmit(ctx, "ai:error", errorMessage)`

### Prompt construction

| Inputs | User message sent to API |
|--------|--------------------------|
| Instructions + selected text | `"{instructions}\n\n{selected_text}"` |
| Selected text only (no instructions) | `"{selected_text}"` |
| Instructions only (no selected text) | `"{instructions}"` |

System prompt: exactly what is configured per provider. No automatic injection.

---

## Result Window

**Appearance:**
- Frameless, always-on-top
- Fixed size: 520 × 420 px (max)
- If response is shorter, window can be smaller — but has a minimum height of ~160 px
- Rounded corners, drop shadow
- Matches popup theme

**Contents:**
```
┌────────────────────────────────────────┐
│                                        │
│  [Spinner / streaming text here]       │
│  (scrollable if content overflows)     │
│                                        │
│                          [Copy][Close] │
└────────────────────────────────────────┘
```

**States:**

1. **Loading** — spinner centered, Copy button disabled
2. **Streaming** — text appears token by token, spinner gone, Copy button enabled (copies partial)
3. **Done** — text complete, no explicit visual change (stream just stops)
4. **Error** — partial text (if any) remains; error banner appears at bottom above buttons: `"Error: {message}"`

**Interaction:**
- **Copy** button: copies full text to clipboard, then auto-closes the result window
- **Close** button: closes result window
- **Escape**: if stream is in progress → cancel stream, keep partial result visible; if stream is done → close window
- **Click outside**: does NOT close (result stays open)

**Cancellation:**
- Escape during streaming cancels the HTTP request context
- Whatever text arrived before cancellation remains displayed
- Copy and Close buttons remain active on partial result

---

## Tray

### Icon states
- **Normal**: simple key icon (no text)
- **Warning** (hotkey conflict): key icon with exclamation badge

### Menu
```
PromptKey
──────────────
● Claude              ← active provider, non-interactive label
  ┌ Switch Provider ▶
  │   ✓ Claude
  │     Ollama
  └─────────────────
Settings
──────────────
Quit
```

- Switching provider is immediate (in-memory + persisted to config); no restart
- If only one provider is configured, the Switch Provider submenu is still shown but has only one item (checked)

### First run (no config)
- App starts, tray icon appears
- Show OS tray notification: `"PromptKey: No provider configured. Click the tray icon to open Settings."`
- Pressing the hotkey while unconfigured opens Settings (same as if hotkey were triggered with no providers)

---

## Settings Window

Standard framed window. Sections: **Providers**, **Hotkey**, **Appearance**.

### Providers section
- List of configured providers (name, model shown)
- **Add** button opens an inline form below the list
- **Edit** button (per row) populates the form for editing
- **Delete** button (per row) removes after confirmation
- **Set Active** button (per row, or click-to-activate) marks a provider as active
- Form fields: Name, Base URL, API Key (masked), Model, System Prompt (textarea)

### Hotkey section
- Text input field: user types the hotkey string (e.g. `ctrl+shift+space`)
- Validated on save; invalid combinations show an inline error
- Current hotkey is re-registered on save without restart

### Appearance section
- Theme: dropdown — `Auto (System)`, `Light`, `Dark`

### Save behavior
- **Save** button writes config to disk
- Changes take effect immediately (no restart required)

---

## Config Schema

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
    }
  ],
  "activeProvider": "Claude"
}
```

**`theme`**: `"auto"` | `"light"` | `"dark"`. Default: `"auto"`.

Config path:
- Windows: `%APPDATA%\promptkey\config.json`
- macOS: `~/Library/Application Support/promptkey/config.json`

Missing fields fall back to defaults. Config is created on first launch if absent.

---

## AI Streaming Client

- Uses OpenAI-compatible `/chat/completions` endpoint with `stream: true`
- Parses SSE `data:` lines, extracts `choices[0].delta.content`
- Passes each chunk to a channel; goroutine emits Wails events from that channel
- Request is cancelled via `context.WithCancel` — cancel func stored on `App`, called on Escape
- Errors (non-2xx response, JSON parse failure, network drop) are emitted as `ai:error`

---

## Theme

- Wails `Theme` config set to `SystemDefault` unless user overrides
- Frontend reads theme preference from config on load
- CSS variables switch between light/dark palettes
- System appearance change re-applies automatically when `auto`

---

## Build Order

Follow this order — each step is a working, shippable increment:

1. **Tray**: icon appears, tray menu with Quit. Nothing else.
2. **Hotkey + popup**: hotkey shows frameless window near cursor. Escape/blur closes. No input yet.
3. **Input + echo**: text input in popup. Enter logs to console. No AI.
4. **Context capture**: clipboard simulation. Paperclip icon shown if context captured. Still echo-only.
5. **AI streaming**: real API call. Result window with spinner → streaming text → Copy/Close.
6. **Settings window**: providers, hotkey, theme. Config persistence. Tray menu switch.
7. **Polish**: error states, warning tray icon, first-run notification, theme switching, edge cases.

---

## Known Limitations

- If selected text is byte-for-byte identical to existing clipboard contents, it is silently treated as no context (safety over false positive)
- Non-text clipboard contents (images, files) are not restored after capture attempt
- Very large selected text is sent as-is; API-side context limit errors surface as error banners
- Clipboard simulation may fail in apps that block synthetic key events (no workaround)

---

## Out of Scope (for now)

- Conversation history / multi-turn
- Image input
- Local model auto-discovery
- macOS build (Phase 2)
- Custom result window sizing
- Hotkey press-to-capture UI

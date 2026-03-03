# CLAUDE.md

## Project

PromptKey — a cross-platform tray utility (macOS + Windows) that captures selected text and opens a floating AI prompt popup near the cursor. See @README.md for full context: architecture, UX flow, config schema, platform notes, and build order.

## Stack

Go + Wails v2 (Svelte frontend).

## Dev Environment

Development runs in **WSL** with a **Nix flake**. Always work inside the dev shell:

```bash
nix develop
```

All tools (Go, Node.js, Wails CLI, etc.) must be declared in `flake.nix`. Never ask the user to install anything manually — if a tool is missing, add it to the flake.

## Commands

```bash
wails dev      # run with hot reload
wails build    # production binary for current platform
go test ./...  # run tests
```

## Git

Commits are atomic — one coherent change per commit. Subject lines are short and imperative. Add context in the body if needed, not the subject.

Claude-authored commits must set the author explicitly:

```bash
git commit --author="Claude <claude@anthropic.com>" -m "subject"
```

## Code

- Return and wrap errors: `fmt.Errorf("context: %w", err)` — never panic
- AI calls run in their own goroutine — never block main
- No global mutable state outside the `App` struct
- Svelte components use scoped `<style>` blocks
- `gofmt` and `go vet` are enforced by a pre-commit hook — do not skip or suppress

## Documentation

Always use the Context7 MCP for library and API documentation, code generation, and setup or configuration steps — without waiting to be asked.

## Philosophy

Use idiomatic solutions — write Go like Go, Svelte like Svelte. Follow the conventions of the language and framework rather than importing patterns from elsewhere.

Use current, stable versions of all dependencies. Prefer stdlib over third-party where stdlib is sufficient.

Do not overengineer. The simplest solution that correctly solves the problem is the right one.

After every implementation, evaluate: is this more complex than the problem warrants? If a simpler approach exists, remake it. If the feature adds complexity without clear value, raise it with the operator — it may be better dropped entirely.

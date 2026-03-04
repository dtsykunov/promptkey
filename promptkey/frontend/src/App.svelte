<script>
  import { onMount, tick } from 'svelte'
  import { EventsOn, Hide } from '../wailsjs/runtime/runtime.js'
  import { SendPrompt, Retry, SaveResultSize } from '../wailsjs/go/main/App.js'

  // View: 'popup' | 'result'
  let view = 'popup'

  // Popup state
  let input
  let text = ''

  // Result state
  let responseText = ''
  let streaming = false
  let errorMsg = ''
  let responseDiv

  function resetResult() {
    responseText = ''
    streaming = false
    errorMsg = ''
  }

  function submit() {
    const instructions = text.trim()
    if (!instructions) return
    text = ''
    SendPrompt(instructions)
    // Go handles resize + reposition + emits result:open
  }

  function closeResult() {
    Hide()
    view = 'popup'
    resetResult()
  }

  async function scrollBottom() {
    await tick()
    if (responseDiv) {
      responseDiv.scrollTop = responseDiv.scrollHeight
    }
  }

  // Debounced resize handler — saves result window size when user resizes it.
  let resizeTimer
  function onWindowResize() {
    if (view !== 'result') return
    clearTimeout(resizeTimer)
    resizeTimer = setTimeout(() => {
      SaveResultSize(window.innerWidth, window.innerHeight)
    }, 500)
  }

  onMount(() => {
    window.addEventListener('resize', onWindowResize)

    EventsOn('popup:open', () => {
      view = 'popup'
      resetResult()
      text = ''
      tick().then(() => input?.focus())
    })

    EventsOn('popup:dismiss', () => {
      Hide()
      view = 'popup'
      resetResult()
      text = ''
    })

    EventsOn('result:open', () => {
      view = 'result'
      resetResult()
      streaming = true
    })

    EventsOn('ai:chunk', (chunk) => {
      responseText += chunk
      scrollBottom()
    })

    EventsOn('ai:done', () => {
      streaming = false
    })

    EventsOn('ai:error', (msg) => {
      streaming = false
      errorMsg = msg
    })
  })
</script>

{#if view === 'popup'}
  <main class="popup">
    <input
      bind:this={input}
      bind:value={text}
      placeholder="Ask anything…"
      on:keydown={(e) => {
        if (e.key === 'Enter')  submit()
        if (e.key === 'Escape') { Hide(); text = '' }
      }}
    />
    <button on:mousedown|preventDefault on:click={submit}>↵</button>
  </main>
{:else}
  <main class="result">
    <!-- Drag handle — the entire bar moves the window -->
    <div class="drag-bar" style="--wails-draggable:drag"></div>

    <div class="response" bind:this={responseDiv}>
      {#if responseText}{responseText}{/if}{#if streaming}<span class="cursor">▋</span>{/if}
    </div>

    {#if errorMsg}
      <div class="error-banner">⚠ {errorMsg}</div>
    {/if}

    <div class="actions">
      <button
        class="action"
        disabled={streaming}
        on:click={() => navigator.clipboard.writeText(responseText)}
      >Copy</button>
      <button
        class="action"
        on:click={() => { resetResult(); streaming = true; Retry() }}
      >Retry</button>
      <button class="action" on:click={closeResult}>Close</button>
    </div>
  </main>
{/if}

<svelte:window on:keydown={(e) => {
  if (view === 'result' && e.key === 'Escape') closeResult()
}} />

<style>
  :global(body) {
    margin: 0;
    overflow: hidden;
  }

  main {
    width: 100%;
    height: 100%;
    background: #1e1e2e;
    box-sizing: border-box;
    font-family: inherit;
    color: #cdd6f4;
  }

  /* ── Popup ── */
  main.popup {
    border-radius: 8px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
    display: flex;
    align-items: center;
    padding: 0 12px;
  }

  input {
    flex: 1;
    background: transparent;
    border: none;
    outline: none;
    color: #cdd6f4;
    font-size: 15px;
    font-family: inherit;
  }

  input::placeholder {
    color: #585b70;
  }

  main.popup button {
    background: transparent;
    border: none;
    color: #585b70;
    cursor: pointer;
    font-size: 16px;
    padding: 0 4px;
    flex-shrink: 0;
  }

  main.popup button:hover {
    color: #cdd6f4;
  }

  /* ── Result ── */
  main.result {
    border-radius: 8px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
    display: flex;
    flex-direction: column;
    padding: 0 16px 16px;
    gap: 10px;
  }

  .drag-bar {
    height: 20px;
    cursor: grab;
    flex-shrink: 0;
    border-radius: 8px 8px 0 0;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .drag-bar:active {
    cursor: grabbing;
  }

  /* Two short horizontal lines as grip indicator */
  .drag-bar::before {
    content: '';
    display: block;
    width: 32px;
    height: 2px;
    background: #45475a;
    border-radius: 2px;
    box-shadow: 0 5px 0 #45475a;
  }

  .response {
    flex: 1;
    overflow-y: auto;
    font-family: 'Consolas', 'Monaco', monospace;
    font-size: 14px;
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-word;
    text-align: left;
    color: #cdd6f4;

    /* Dark scrollbar */
    scrollbar-width: thin;
    scrollbar-color: #45475a #1e1e2e;
  }

  .response::-webkit-scrollbar {
    width: 6px;
  }

  .response::-webkit-scrollbar-track {
    background: #1e1e2e;
  }

  .response::-webkit-scrollbar-thumb {
    background: #45475a;
    border-radius: 3px;
  }

  .response::-webkit-scrollbar-thumb:hover {
    background: #585b70;
  }

  .cursor {
    display: inline-block;
    animation: blink 1s step-start infinite;
    color: #89b4fa;
  }

  @keyframes blink {
    50% { opacity: 0; }
  }

  .error-banner {
    background: #3b1f1f;
    border: 1px solid #f38ba8;
    border-radius: 6px;
    color: #f38ba8;
    font-size: 13px;
    padding: 8px 12px;
  }

  .actions {
    display: flex;
    gap: 8px;
    justify-content: center;
    flex-shrink: 0;
  }

  .action {
    background: #313244;
    border: 1px solid #45475a;
    border-radius: 6px;
    color: #cdd6f4;
    cursor: pointer;
    font-size: 13px;
    padding: 6px 16px;
    transition: background 0.15s;
  }

  .action:hover:not(:disabled) {
    background: #45475a;
  }

  .action:disabled {
    color: #585b70;
    cursor: default;
  }
</style>

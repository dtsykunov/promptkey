<script>
  import { onMount, tick } from 'svelte'
  import { EventsOn, Hide } from '../wailsjs/runtime/runtime.js'
  import { SendPrompt, Retry, SaveResultSize, GetConfig, SaveSettings, FetchModels } from '../wailsjs/go/main/App.js'

  // View: 'popup' | 'result' | 'settings'
  let view = 'popup'

  // Popup state
  let input
  let text = ''

  // Result state
  let responseText = ''
  let streaming = false
  let errorMsg = ''
  let responseDiv

  // Settings state
  let localCfg = null        // working copy; populated on settings:open
  let settingsPanel = 'list' // 'list' | 'edit'
  let editIdx = -1           // index in localCfg.providers; -1 = new
  let editProvider = {}
  let modelList = []
  let fetchState = null      // null | 'loading' | 'ok' | 'error'
  let fetchError = ''
  let settingsError = ''

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

  let resizeTimer
  function onWindowResize() {
    if (view !== 'result') return
    clearTimeout(resizeTimer)
    resizeTimer = setTimeout(() => {
      SaveResultSize(window.innerWidth, window.innerHeight)
    }, 500)
  }

  // ── Settings helpers ──────────────────────────────────────────

  function openEditPanel(idx) {
    editIdx = idx
    if (idx === -1) {
      editProvider = { name: '', baseURL: '', apiKey: '', model: '', systemPrompt: '' }
    } else {
      editProvider = { ...localCfg.providers[idx] }
    }
    modelList = []
    fetchState = null
    fetchError = ''
    settingsPanel = 'edit'
  }

  function removeProvider(idx) {
    localCfg.providers = localCfg.providers.filter((_, i) => i !== idx)
    if (localCfg.activeProvider === localCfg.providers[idx]?.name) {
      localCfg.activeProvider = localCfg.providers[0]?.name ?? ''
    }
  }

  function setActive(name) {
    localCfg.activeProvider = name
  }

  async function tryFetchModels() {
    if (!editProvider.baseURL) return
    fetchState = 'loading'
    fetchError = ''
    modelList = []
    try {
      const ids = await FetchModels(editProvider.baseURL, editProvider.apiKey || '')
      modelList = ids
      fetchState = 'ok'
    } catch (e) {
      fetchError = String(e)
      fetchState = 'error'
    }
  }

  function confirmEdit() {
    if (!editProvider.name.trim()) return
    if (editIdx === -1) {
      localCfg.providers = [...localCfg.providers, { ...editProvider }]
    } else {
      localCfg.providers = localCfg.providers.map((p, i) => i === editIdx ? { ...editProvider } : p)
    }
    settingsPanel = 'list'
  }

  function cancelEdit() {
    settingsPanel = 'list'
  }

  async function saveSettings() {
    settingsError = ''
    const err = await SaveSettings(localCfg)
    if (err !== '') {
      settingsError = err
      return
    }
    Hide()
    view = 'popup'
  }

  function cancelSettings() {
    Hide()
    view = 'popup'
  }

  // ─────────────────────────────────────────────────────────────

  onMount(() => {
    window.addEventListener('resize', onWindowResize)
    window.addEventListener('contextmenu', e => e.preventDefault())

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

    EventsOn('settings:open', async () => {
      view = 'settings'
      settingsPanel = 'list'
      settingsError = ''
      localCfg = await GetConfig()
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

{:else if view === 'result'}
  <main class="result">
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

{:else if view === 'settings' && localCfg}
  <main class="settings">
    <div class="settings-titlebar" style="--wails-draggable:drag">
      <span class="settings-title">Settings</span>
      <button class="btn-close" on:click={cancelSettings} style="--wails-draggable:no-drag">✕</button>
    </div>
    {#if settingsPanel === 'list'}
      <div class="settings-body">
        <div class="field-row">
          <label for="hotkey-input">Hotkey</label>
          <input
            id="hotkey-input"
            class="text-input"
            bind:value={localCfg.hotkey}
            placeholder={`ctrl+alt+\``}
          />
        </div>

        <div class="section-header">
          <span>Providers</span>
          <button class="btn-small" on:click={() => openEditPanel(-1)}>+ Add</button>
        </div>

        <div class="provider-list">
          {#each localCfg.providers as p, i (i)}
            <div
              class="provider-row"
              class:active={localCfg.activeProvider === p.name}
              on:click={() => setActive(p.name)}
              on:keydown={(e) => e.key === 'Enter' && setActive(p.name)}
              role="button"
              tabindex="0"
            >
              <span class="dot">{localCfg.activeProvider === p.name ? '●' : '○'}</span>
              <span class="provider-name">{p.name || '(unnamed)'}</span>
              <button class="btn-icon" title="Edit" on:click|stopPropagation={() => openEditPanel(i)}>✎</button>
              <button class="btn-icon danger" title="Remove" on:click|stopPropagation={() => removeProvider(i)}>✕</button>
            </div>
          {/each}
        </div>

        {#if settingsError}
          <div class="error-banner">⚠ {settingsError}</div>
        {/if}
      </div>

      <div class="settings-footer">
        <button class="action" on:click={cancelSettings}>Close</button>
        <button class="action primary" on:click={saveSettings}>Save</button>
      </div>

    {:else}
      <!-- Edit panel -->
      <div class="settings-body">
        <button class="btn-back" on:click={cancelEdit}>← Back</button>

        <div class="field-row">
          <label>Name</label>
          <input class="text-input" bind:value={editProvider.name} placeholder="My Provider" />
        </div>

        <div class="field-row">
          <label>Base URL</label>
          <div class="input-with-status">
            <input
              class="text-input"
              bind:value={editProvider.baseURL}
              placeholder="https://api.openai.com/v1"
              on:blur={tryFetchModels}
            />
            {#if fetchState === 'loading'}<span class="status-icon spin">⟳</span>
            {:else if fetchState === 'ok'}<span class="status-icon ok">✓</span>
            {:else if fetchState === 'error'}<span class="status-icon err" title={fetchError}>✗</span>
            {/if}
          </div>
        </div>

        <div class="field-row">
          <label>API Key</label>
          <input
            class="text-input"
            type="password"
            bind:value={editProvider.apiKey}
            placeholder="sk-..."
            on:blur={tryFetchModels}
          />
        </div>

        <div class="field-row">
          <label>Model</label>
          {#if modelList.length > 0}
            <select class="text-input" bind:value={editProvider.model}>
              {#if editProvider.model && !modelList.includes(editProvider.model)}
                <option value={editProvider.model}>{editProvider.model}</option>
              {/if}
              {#each modelList as m}
                <option value={m}>{m}</option>
              {/each}
            </select>
          {:else}
            <input class="text-input" bind:value={editProvider.model} placeholder="gpt-4o" />
          {/if}
        </div>

        <div class="field-col">
          <label>System prompt</label>
          <textarea
            class="text-input textarea"
            bind:value={editProvider.systemPrompt}
            placeholder="You are a concise, helpful assistant."
          ></textarea>
        </div>

        {#if fetchState === 'error'}
          <div class="error-banner">⚠ {fetchError}</div>
        {/if}
      </div>

      <div class="settings-footer">
        <button class="action" on:click={cancelEdit}>Cancel</button>
        <button class="action primary" disabled={!editProvider.name.trim()} on:click={confirmEdit}>OK</button>
      </div>
    {/if}
  </main>
{/if}

<svelte:window on:keydown={(e) => {
  if (view === 'result' && e.key === 'Escape') closeResult()
  if (view === 'settings' && e.key === 'Escape') {
    if (settingsPanel === 'edit') cancelEdit()
    else cancelSettings()
  }
}} />

<style>
  :global(body) {
    margin: 0;
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
    overflow: hidden;
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
    overflow: hidden;
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
    scrollbar-width: thin;
    scrollbar-color: #45475a #1e1e2e;
  }

  .response::-webkit-scrollbar { width: 6px; }
  .response::-webkit-scrollbar-track { background: #1e1e2e; }
  .response::-webkit-scrollbar-thumb { background: #45475a; border-radius: 3px; }
  .response::-webkit-scrollbar-thumb:hover { background: #585b70; }

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

  .action.primary {
    background: #89b4fa;
    border-color: #89b4fa;
    color: #1e1e2e;
    font-weight: 600;
  }

  .action.primary:hover:not(:disabled) {
    background: #b4d0ff;
  }

  /* ── Settings ── */
  main.settings {
    display: flex;
    flex-direction: column;
    height: 100%;
    padding: 0;
  }

  .settings-titlebar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 12px 0 20px;
    height: 40px;
    flex-shrink: 0;
    cursor: grab;
    border-bottom: 1px solid #313244;
  }

  .settings-titlebar:active {
    cursor: grabbing;
  }

  .settings-title {
    font-size: 13px;
    font-weight: 600;
    color: #a6adc8;
    pointer-events: none;
  }

  .btn-close {
    background: transparent;
    border: none;
    color: #585b70;
    cursor: pointer;
    font-size: 14px;
    padding: 4px 6px;
    border-radius: 4px;
    line-height: 1;
  }

  .btn-close:hover {
    color: #f38ba8;
    background: #3b1f1f;
  }

  .settings-body {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding: 20px 24px 8px;
    display: block;
    scrollbar-width: thin;
    scrollbar-color: #45475a #1e1e2e;
  }

  /* Spacing between form sections in block layout */
  .field-row,
  .field-col,
  .section-header,
  .provider-list,
  .btn-back,
  .error-banner {
    margin-bottom: 14px;
  }

  .settings-footer {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
    padding: 12px 24px;
    border-top: 1px solid #313244;
    flex-shrink: 0;
  }

  .field-row {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .field-row label {
    width: 80px;
    font-size: 13px;
    color: #a6adc8;
    flex-shrink: 0;
  }

  .field-col {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .field-col label {
    font-size: 13px;
    color: #a6adc8;
  }

  .text-input {
    flex: 1;
    background: #313244;
    border: 1px solid #45475a;
    border-radius: 6px;
    color: #cdd6f4;
    font-size: 13px;
    font-family: inherit;
    padding: 6px 10px;
    outline: none;
    width: 100%;
    box-sizing: border-box;
  }

  .text-input:focus {
    border-color: #89b4fa;
  }

  .textarea {
    flex: none;
    resize: vertical;
    height: 100px;
    min-height: 60px;
    overflow: auto;
  }

  select.text-input {
    cursor: pointer;
  }

  .input-with-status {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .input-with-status .text-input {
    flex: 1;
  }

  .status-icon {
    font-size: 14px;
    flex-shrink: 0;
    width: 18px;
    text-align: center;
  }

  .status-icon.ok { color: #a6e3a1; }
  .status-icon.err { color: #f38ba8; }
  .status-icon.spin {
    color: #89b4fa;
    display: inline-block;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 12px;
    font-weight: 600;
    color: #a6adc8;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    border-bottom: 1px solid #313244;
    padding-bottom: 6px;
  }

  .btn-small {
    background: #313244;
    border: 1px solid #45475a;
    border-radius: 5px;
    color: #cdd6f4;
    cursor: pointer;
    font-size: 12px;
    padding: 3px 10px;
  }

  .btn-small:hover { background: #45475a; }

  .provider-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .provider-row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 10px;
    border-radius: 6px;
    cursor: pointer;
    user-select: none;
    background: #313244;
  }

  .provider-row:hover {
    background: #363649;
  }

  .provider-row.active {
    background: #2a2a3e;
    border: 1px solid #89b4fa44;
  }

  .dot {
    font-size: 11px;
    color: #89b4fa;
    flex-shrink: 0;
  }

  .provider-name {
    flex: 1;
    font-size: 13px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .btn-icon {
    background: transparent;
    border: none;
    color: #585b70;
    cursor: pointer;
    font-size: 14px;
    padding: 2px 4px;
    border-radius: 4px;
    line-height: 1;
  }

  .btn-icon:hover { color: #cdd6f4; background: #45475a; }
  .btn-icon.danger:hover { color: #f38ba8; background: #3b1f1f; }

  .btn-back {
    background: transparent;
    border: none;
    color: #89b4fa;
    cursor: pointer;
    font-size: 13px;
    padding: 0;
    text-align: left;
    margin-bottom: 4px;
  }

  .btn-back:hover { text-decoration: underline; }
</style>

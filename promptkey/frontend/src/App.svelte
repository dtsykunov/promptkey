<script>
  import { onMount } from 'svelte'
  import { EventsOn, Hide } from '../wailsjs/runtime/runtime.js'
  import { SendPrompt } from '../wailsjs/go/main/App.js'

  let input
  let text = ''
  let ready = false
  let contextCaptured = false

  function submit() {
    ready = false
    contextCaptured = false
    SendPrompt(text.trim())
    text = ''
    Hide()
  }

  function dismiss() {
    ready = false
    contextCaptured = false
    text = ''
    Hide()
  }

  onMount(() => {
    EventsOn('popup:open', (hasContext) => {
      ready = true
      contextCaptured = !!hasContext
      input?.focus()
    })
    EventsOn('wails:window:unfocus', () => {
      if (ready) dismiss()
    })
  })
</script>

<main>
  {#if contextCaptured}
    <svg class="paperclip" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"
         width="16" height="16" fill="none" stroke="currentColor"
         stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M21.44 11.05l-9.19 9.19a6 6 0 01-8.49-8.49l9.19-9.19a4 4 0 015.66 5.66l-9.2 9.19a2 2 0 01-2.83-2.83l8.49-8.48"/>
    </svg>
  {/if}
  <input
    bind:this={input}
    bind:value={text}
    placeholder="Ask anything…"
    on:keydown={(e) => {
      if (e.key === 'Enter')  submit()
      if (e.key === 'Escape') dismiss()
    }}
  />
  <button on:mousedown|preventDefault on:click={submit}>↵</button>
</main>

<style>
  main {
    width: 100%;
    height: 100%;
    background: #1e1e2e;
    border-radius: 8px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
    display: flex;
    align-items: center;
    padding: 0 12px;
    box-sizing: border-box;
  }

  .paperclip {
    color: #585b70;
    flex-shrink: 0;
    margin-right: 6px;
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

  button {
    background: transparent;
    border: none;
    color: #585b70;
    cursor: pointer;
    font-size: 16px;
    padding: 0 4px;
    flex-shrink: 0;
  }

  button:hover {
    color: #cdd6f4;
  }
</style>

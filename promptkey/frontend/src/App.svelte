<script>
  import { onMount } from 'svelte'
  import { EventsOn, Hide } from '../wailsjs/runtime/runtime.js'
  import { SendPrompt } from '../wailsjs/go/main/App.js'

  let input
  let text = ''
  let ready = false

  function submit() {
    ready = false
    SendPrompt(text.trim())
    text = ''
    Hide()
  }

  function dismiss() {
    ready = false
    text = ''
    Hide()
  }

  onMount(() => {
    EventsOn('popup:open', () => {
      ready = true
      input?.focus()
    })
    EventsOn('popup:dismiss', dismiss)
  })
</script>

<main>
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

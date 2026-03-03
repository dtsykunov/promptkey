<script>
  import { onMount } from 'svelte'
  import { EventsOn, Hide, LogPrint } from '../wailsjs/runtime/runtime.js'

  let input
  let text = ''
  let ready = false

  function submit() {
    ready = false
    if (text.trim()) LogPrint(text.trim())
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
    window.addEventListener('blur', () => {
      if (ready) dismiss()
    })
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
    width: 100%;
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
</style>

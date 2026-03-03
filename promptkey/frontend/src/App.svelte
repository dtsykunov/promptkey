<script>
  import { onMount } from 'svelte'
  import { Hide } from '../../wailsjs/runtime/runtime.js'

  onMount(() => {
    window.addEventListener('blur', () => Hide())
    window.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') Hide()
    })
  })
</script>

<main></main>

<style>
  main {
    width: 100%;
    height: 100%;
    background: #1e1e2e;
    border-radius: 8px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  }
</style>

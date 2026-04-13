<script>
  import MainMenu from './MainMenu.svelte'
  import Login from './Login.svelte'
  import NetworkMenu from './NetworkMenu.svelte'
  import Dashboard from './Dashboard.svelte'
  import About from './About.svelte'
  import { lang, toggleLang } from './lang.js'

  let screen = 'main'
  let prev = 'main'
  function go(to) { prev = screen; screen = to }
  function back() { screen = prev }
</script>

<div class="root">
  <button class="lang-btn" on:click={toggleLang} title={$lang === 'hr' ? 'Switch to English' : 'Prebaci na hrvatski'}>
    <img src="/flags/{$lang === 'hr' ? 'hr' : 'gb'}.png" alt={$lang} class="flag-img" />
  </button>

  {#key screen}
    {#if screen === 'main'}
      <MainMenu onLogin={() => go('login')} onAbout={() => go('about')} />
    {:else if screen === 'login'}
      <Login onSuccess={() => go('network')} onBack={() => go('main')} />
    {:else if screen === 'network'}
      <NetworkMenu onConnected={() => go('dashboard')} onAbout={() => go('about')} />
    {:else if screen === 'dashboard'}
      <Dashboard onDisconnect={() => go('network')} />
    {:else if screen === 'about'}
      <About onBack={back} />
    {/if}
  {/key}
</div>

<style>
  .root {
    width: 100vw; height: 100vh;
    overflow: hidden;
    display: flex; justify-content: center; align-items: center;
    background: radial-gradient(ellipse at 50% 40%, #0e1a0e 0%, #080d08 40%, #040604 100%);
    position: relative;
  }

  .lang-btn {
    position: fixed; top: 8px; right: 10px; z-index: 100;
    background: rgba(14,21,14,0.9); border: 1px solid #2a3d2a;
    border-radius: 4px; padding: 3px 5px;
    cursor: pointer; transition: all 0.2s; line-height: 0;
  }
  .lang-btn:hover { background: rgba(42,74,42,0.6); transform: scale(1.1); }

  .flag-img { width: 28px; height: 18px; border-radius: 2px; display: block; }
</style>

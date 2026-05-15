<script>
  import MainMenu from './MainMenu.svelte'
  import Login from './Login.svelte'
  import NetworkMenu from './NetworkMenu.svelte'
  import Dashboard from './Dashboard.svelte'
  import About from './About.svelte'
  import Settings from './Settings.svelte'
  import { lang, toggleLang } from './lang.js'
  import { GetHrFlag, GetGbFlag } from '../wailsjs/go/main/App.js'
  import { onMount } from 'svelte'

  let screen = 'main'
  let history = []
  let hrFlag = ''
  let gbFlag = ''

  onMount(async () => {
    hrFlag = await GetHrFlag()
    gbFlag = await GetGbFlag()
  })

  function go(to) { history = [...history, screen]; screen = to }
  function back() { screen = history.pop() || 'main'; history = history }

  $: currentFlag = $lang === 'hr' ? hrFlag : gbFlag
</script>

<div class="root">
  <button class="lang-btn" on:click={toggleLang} title={$lang === 'hr' ? 'Switch to English' : 'Prebaci na hrvatski'}>
    {#if currentFlag}
      <img src={currentFlag} alt={$lang} class="flag-img" />
    {:else}
      <span style="font-size:.8em;color:#c9b06b">{$lang.toUpperCase()}</span>
    {/if}
  </button>

  {#key screen}
    {#if screen === 'main'}
      <MainMenu onLogin={() => go('login')} onAbout={() => go('about')} onSettings={() => go('settings')} />
    {:else if screen === 'login'}
      <Login onSuccess={() => go('network')} onBack={back} />
    {:else if screen === 'network'}
      <NetworkMenu onConnected={() => go('dashboard')} onAbout={() => go('about')} onBack={() => { screen='main'; history=[] }} />
    {:else if screen === 'dashboard'}
      <Dashboard onBack={() => go('network')} onLeave={() => { screen='network'; history=[] }} />
    {:else if screen === 'about'}
      <About onBack={back} />
    {:else if screen === 'settings'}
      <Settings onBack={back} />
    {/if}
  {/key}
</div>

<style>
  .root{width:100vw;height:100vh;overflow:hidden;display:flex;justify-content:center;align-items:center;background:radial-gradient(ellipse at 50% 40%,#0e1a0e 0%,#080d08 40%,#040604 100%);position:relative}
  .lang-btn{position:fixed;top:8px;right:10px;z-index:100;background:rgba(14,21,14,.9);border:1px solid #2a3d2a;border-radius:4px;padding:3px 5px;cursor:pointer;transition:all .2s;line-height:0;min-width:36px;min-height:24px;display:flex;align-items:center;justify-content:center}
  .lang-btn:hover{background:rgba(42,74,42,.6);transform:scale(1.1)}
  .flag-img{width:28px;height:18px;border-radius:2px;display:block}
</style>

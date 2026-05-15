<script>
  import { GetServerURL, SetServerURL } from '../wailsjs/go/main/App.js'
  import { lang, ui } from './lang.js'
  import { onMount } from 'svelte'
  export let onBack

  $: t = ui[$lang]
  let serverUrl = ''
  let saved = false

  onMount(async () => { serverUrl = await GetServerURL() })

  function save() {
    SetServerURL(serverUrl)
    saved = true
    setTimeout(() => saved = false, 1500)
  }
</script>

<div class="panel">
  <h1 class="title">Pa<span class="lan">LAN</span>tir</h1>
  <p class="sub">{t.settingsTitle}</p>
  <div class="form">
    <label class="lbl">{t.serverUrl}</label>
    <input type="text" bind:value={serverUrl} placeholder="http://IP:8080" />
    {#if saved}<div class="toast ok">✓</div>{/if}
    <button class="btn primary" on:click={save}>{t.save}</button>
    <button class="btn back" on:click={onBack}>{t.back}</button>
  </div>
</div>

<style>
  .panel{background:linear-gradient(170deg,#111a11,#0b120b);border:1px solid #2a3d2a;border-radius:12px;padding:30px 26px;width:420px;text-align:center;box-shadow:0 24px 70px rgba(0,0,0,.5);overflow:hidden}
  .title{font-family:'Cinzel',serif;font-size:2.2em;color:#c9a84c;letter-spacing:3px} .lan{color:#d4c88a;font-weight:900}
  .sub{color:#6b7b4a;font-style:italic;margin:3px 0 20px;font-size:.9em}
  .form{display:flex;flex-direction:column;gap:10px}
  .lbl{color:#6b7b4a;font-size:.85em;text-align:left}
  input{display:block;width:100%;padding:11px 14px;font-family:'Crimson Text',serif;font-size:1em;border:1px solid #2a3d2a;border-radius:6px;background:#152015;color:#d4c88a;outline:none;box-sizing:border-box}
  input:focus{border-color:#5a8a3a}
  .toast.ok{background:#1a2e14;border:1px solid #3a6b2a;color:#a0e8a0;padding:6px;border-radius:5px;font-size:.85em}
  .btn{padding:11px 14px;font-family:'Cinzel',serif;font-size:.9em;font-weight:600;border:1px solid;border-radius:6px;cursor:pointer;transition:all .25s;box-sizing:border-box}
  .btn:hover{transform:translateY(-2px)}
  .primary{background:linear-gradient(180deg,#1a2e1a,#142414);color:#c9b06b;border-color:#3a5a3a}
  .primary:hover{background:#243824;border-color:#5a8a3a}
  .back{background:transparent;color:#6b7b4a;border-color:#2a3d2a;margin-top:3px} .back:hover{color:#c9b06b;border-color:#3a5a3a}
</style>

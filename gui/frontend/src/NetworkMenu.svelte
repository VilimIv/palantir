<script>
  import { DoCreateNetwork, DoJoinNetwork, DoStartTunnel } from '../wailsjs/go/main/App.js'
  import { Quit } from '../wailsjs/runtime/runtime.js'
  import { lang, ui } from './lang.js'
  export let onConnected, onAbout
  $: t = ui[$lang]
  let mode='menu',networkCode='',joinCode='',error='',loading=false

  async function create(){error='';loading=true;try{const r=await DoCreateNetwork();networkCode=r.networkID;mode='created'}catch(e){error=String(e)}loading=false}
  async function startTun(){error='';loading=true;try{await DoStartTunnel();onConnected()}catch(e){error=String(e)}loading=false}
  async function join(){if(!joinCode)return;error='';loading=true;try{await DoJoinNetwork(joinCode);await DoStartTunnel();onConnected()}catch(e){error=String(e)}loading=false}
</script>

<div class="panel">
  <h1 class="title">Pa<span class="lan">LAN</span>tir</h1>
  <div class="orb-mini"></div>
  {#if mode==='menu'}
    <div class="buttons">
      <button class="btn primary" on:click={create} disabled={loading}>{loading?t.creating:t.createNet}</button>
      <button class="btn primary" on:click={()=>mode='join'} disabled={loading}>{t.joinNet}</button>
      <button class="btn primary" on:click={onAbout}>{t.aboutBtn}</button>
      <button class="btn danger" on:click={()=>Quit()}>{t.exitBtn}</button>
    </div>
  {:else if mode==='created'}
    <div class="section">
      <p class="label">{t.netCreated}</p>
      <div class="code-box">{networkCode}</div>
      <p class="hint">{t.shareCode}</p>
      <button class="btn primary" on:click={startTun} disabled={loading}>{loading?t.starting:t.startTunnel}</button>
      <button class="btn back" on:click={()=>mode='menu'}>{t.back}</button>
    </div>
  {:else if mode==='join'}
    <div class="section">
      <p class="label">{t.enterCode}</p>
      <input type="text" placeholder={t.codePlaceholder} bind:value={joinCode} on:keydown={e=>e.key==='Enter'&&join()} disabled={loading} style="text-align:center;letter-spacing:4px;font-size:1.15em"/>
      <button class="btn primary" on:click={join} disabled={loading||!joinCode}>{loading?t.connecting:t.join}</button>
      <button class="btn back" on:click={()=>mode='menu'}>{t.back}</button>
    </div>
  {/if}
  {#if error}<div class="toast err">{error}</div>{/if}
</div>

<style>
  .panel{background:linear-gradient(170deg,#111a11,#0b120b);border:1px solid #2a3d2a;border-radius:12px;padding:28px 26px;width:420px;text-align:center;box-shadow:0 24px 70px rgba(0,0,0,.5)}
  .title{font-family:'Cinzel',serif;font-size:2.2em;color:#c9a84c;letter-spacing:3px} .lan{color:#d4c88a;font-weight:900}
  .orb-mini{width:16px;height:16px;border-radius:50%;margin:5px auto 18px;background:radial-gradient(circle at 35% 30%,#1a3040,#0a1520);box-shadow:0 0 10px rgba(30,70,100,.5);animation:gl 3.5s ease-in-out infinite}
  @keyframes gl{0%,100%{box-shadow:0 0 10px rgba(30,70,100,.5)}50%{box-shadow:0 0 18px rgba(40,90,130,.6)}}
  .buttons,.section{display:flex;flex-direction:column;gap:12px}
  .label{color:#6b7b4a;margin:0} .hint{color:#4a5a3a;font-size:.82em;font-style:italic;margin:0}
  .code-box{font-family:'Cinzel',serif;font-size:1.8em;color:#d4c88a;background:#152015;border:1px solid #3a5a3a;border-radius:8px;padding:12px;letter-spacing:6px}
  input{padding:11px 14px;font-family:'Cinzel',serif;border:1px solid #2a3d2a;border-radius:6px;background:#152015;color:#d4c88a;outline:none;width:100%;transition:all .3s}
  input:focus{border-color:#5a8a3a} input::placeholder{color:#4a5a3a}
  .toast.err{background:#2e0e0e;border:1px solid #8b2a2a;color:#e8a0a0;padding:8px 12px;border-radius:5px;margin-top:6px;font-size:.85em;animation:shake .4s ease-out}
  @keyframes shake{0%{transform:translateX(-8px);opacity:0}25%{transform:translateX(6px)}50%{transform:translateX(-3px)}100%{transform:translateX(0);opacity:1}}
  .btn{padding:12px 18px;font-family:'Cinzel',serif;font-size:.95em;font-weight:600;border:1px solid;border-radius:7px;cursor:pointer;transition:all .25s;letter-spacing:1px}
  .btn:hover:not(:disabled){transform:translateY(-2px);box-shadow:0 5px 18px rgba(0,0,0,.3)} .btn:disabled{opacity:.4;cursor:not-allowed}
  .primary{background:linear-gradient(180deg,#1a2e1a,#142414);color:#c9b06b;border-color:#3a5a3a}
  .primary:hover:not(:disabled){background:#243824;border-color:#5a8a3a}
  .danger{background:#4a1414;color:#d8a0a0;border-color:#6b2a2a} .danger:hover{background:#5c1a1a}
  .back{background:transparent;color:#6b7b4a;border-color:#2a3d2a} .back:hover{color:#c9b06b;border-color:#3a5a3a}
</style>

<script>
  import { onMount, onDestroy } from 'svelte'
  import { DoLeaveNetwork, DoGetPeers, GetVirtualIP, GetNetworkID, GetUsername } from '../wailsjs/go/main/App.js'
  import { EventsOn } from '../wailsjs/runtime/runtime.js'
  import { lang, ui } from './lang.js'

  export let onBack  // vrati na NetworkMenu (ostaje u mreži)
  export let onLeave // napusti mrežu i vrati na NetworkMenu

  $: t = ui[$lang]

  let peers=[], logs=[], virtualIP='', networkID='', username='', status='connecting', logEl
  let cleanups=[]

  onMount(async () => {
    virtualIP = await GetVirtualIP()
    networkID = await GetNetworkID()
    username = await GetUsername()
    peers = await DoGetPeers() || []
    cleanups.push(EventsOn('log', m => {
      logs=[...logs.slice(-200),m]
      setTimeout(()=>{if(logEl)logEl.scrollTop=logEl.scrollHeight},10)
    }))
    cleanups.push(EventsOn('peers-updated', p => { peers=p||[] }))
    cleanups.push(EventsOn('status-changed', s => { status=s }))
  })
  onDestroy(() => cleanups.forEach(c=>c()))

  async function refreshPeers() { peers = await DoGetPeers() || [] }

  async function leave() { await DoLeaveNetwork(); onLeave() }
</script>

<div class="panel">
  <div class="header">
    <button class="link" on:click={onBack}>{t.backToMenu}</button>
    <h2 class="title">Pa<span class="lan">LAN</span>tir</h2>
    <div class="badge-status" class:on={status==='connected'} class:wait={status==='connecting'}>
      <div class="dot"></div>
      {status==='connected'?t.connected:status==='connecting'?t.connectingStatus:t.disconnected}
    </div>
  </div>

  <div class="info">
    <div class="cell"><span class="lbl">{t.user}</span><span class="val">{username}</span></div>
    <div class="cell"><span class="lbl">{t.virtualIP}</span><span class="val mono">{virtualIP}</span></div>
    <div class="cell"><span class="lbl">{t.netCode}</span><span class="val mono gold">{networkID}</span></div>
  </div>

  <div class="sec">
    <div class="sec-header">
      <h3 class="sec-title">{t.peers} <span class="cnt">{peers.length}</span></h3>
      <button class="refresh-btn" on:click={refreshPeers} title="Refresh">{t.refreshPeers}</button>
    </div>
    <div class="peer-list">
      {#if peers.length===0}
        <div class="empty">{t.waitingPeers}</div>
      {/if}
      {#each peers as p}
        <div class="peer">
          <div class="pl"><span class="pdot" class:on={p.ready}></span><span class="pn">{p.username}</span></div>
          <div class="pr"><span class="pip">{p.virtualIP}</span><span class="mode" class:p2p={p.mode==='P2P'} class:relay={p.mode==='RELAY'}>{p.mode}</span></div>
        </div>
      {/each}
    </div>
  </div>

  <div class="sec">
    <h3 class="sec-title">{t.log}</h3>
    <div class="logbox" bind:this={logEl}>
      {#each logs as l}<div class="logln">{l}</div>{/each}
      {#if !logs.length}<div class="logln dim">{t.waitingActivity}</div>{/if}
    </div>
  </div>

  <button class="btn danger" on:click={leave}>{t.disconnect}</button>
</div>

<style>
  .panel{background:linear-gradient(170deg,#111a11,#0b120b);border:1px solid #2a3d2a;border-radius:12px;padding:14px 18px;width:420px;height:600px;display:flex;flex-direction:column;box-shadow:0 24px 70px rgba(0,0,0,.5);overflow:hidden}

  .header{display:flex;justify-content:space-between;align-items:center;margin-bottom:8px;gap:8px}
  .link{background:none;border:none;color:#5a8a3a;cursor:pointer;font-size:.75em;padding:0;white-space:nowrap} .link:hover{text-decoration:underline}
  .title{font-family:'Cinzel',serif;font-size:1.3em;color:#c9a84c;letter-spacing:2px;flex:1;text-align:center} .lan{color:#d4c88a;font-weight:900}

  .badge-status{display:flex;align-items:center;gap:4px;font-size:.65em;color:#6b7b4a;padding:2px 7px;border:1px solid #2a3d2a;border-radius:8px;background:#0e150e;white-space:nowrap}
  .dot{width:6px;height:6px;border-radius:50%;background:#444;transition:all .3s}
  .badge-status.on .dot{background:#5a8a3c;box-shadow:0 0 5px rgba(90,138,60,.6)} .badge-status.on{color:#8aaa6a;border-color:#2a4a2a}
  .badge-status.wait .dot{background:#b8960c;animation:blink 1s infinite} .badge-status.wait{color:#c9b06b}
  @keyframes blink{50%{opacity:.3}}

  .info{display:flex;gap:5px;background:rgba(100,160,60,.03);border:1px solid #1a2e1a;border-radius:6px;padding:7px 8px;margin-bottom:8px}
  .cell{flex:1;display:flex;flex-direction:column;align-items:center}
  .lbl{color:#4a5a3a;font-size:.6em;text-transform:uppercase;letter-spacing:1px} .val{color:#c9b06b;font-weight:600;font-size:.8em;margin-top:1px}
  .mono{font-family:'Consolas',monospace;letter-spacing:1px} .gold{text-shadow:0 0 5px rgba(201,176,107,.2)}

  .sec{margin-bottom:6px;flex-shrink:0}
  .sec-header{display:flex;justify-content:space-between;align-items:center}
  .sec-title{color:#6b7b4a;font-size:.7em;text-transform:uppercase;letter-spacing:2px;font-family:'Cinzel',serif;margin:0 0 4px} .cnt{color:#c9a84c}
  .refresh-btn{background:none;border:1px solid #2a3d2a;border-radius:4px;color:#6b7b4a;cursor:pointer;font-size:.9em;padding:1px 6px;transition:all .2s} .refresh-btn:hover{color:#c9b06b;border-color:#3a5a3a}

  .peer-list{display:flex;flex-direction:column;gap:3px;max-height:100px;overflow-y:auto}
  .empty{color:#4a5a3a;font-style:italic;text-align:center;padding:10px;font-size:.85em}
  .peer{display:flex;justify-content:space-between;align-items:center;background:rgba(100,160,60,.03);border:1px solid #1a2e1a;border-radius:4px;padding:5px 8px}
  .peer:hover{background:rgba(100,160,60,.06)}
  .pl{display:flex;align-items:center;gap:6px} .pr{display:flex;align-items:center;gap:6px}
  .pdot{width:6px;height:6px;border-radius:50%;background:#444} .pdot.on{background:#5a8a3c;box-shadow:0 0 4px rgba(90,138,60,.5)}
  .pn{color:#c9b06b;font-weight:600;font-size:.85em} .pip{color:#6b7b4a;font-family:monospace;font-size:.7em}
  .mode{padding:1px 6px;border-radius:6px;font-size:.55em;font-weight:700;letter-spacing:1px}
  .mode.p2p{background:#1a2e14;color:#8aaa6a;border:1px solid #2a4a2a} .mode.relay{background:#2e2210;color:#c8a050;border:1px solid #4a3a1a}

  .logbox{background:#060a06;border:1px solid #1a2e1a;border-radius:4px;padding:5px 7px;flex:1;min-height:60px;overflow-y:auto;overflow-x:hidden;font-family:'Consolas',monospace;font-size:.62em}
  .logln{color:#5a7a4a;padding:1px 0;word-break:break-all} .logln.dim{color:#2a3d2a}

  .sec:last-of-type{flex:1;display:flex;flex-direction:column;min-height:0}

  .btn{width:100%;padding:8px;font-family:'Cinzel',serif;font-size:.85em;font-weight:600;border:1px solid;border-radius:6px;cursor:pointer;transition:all .25s;letter-spacing:1px;flex-shrink:0;box-sizing:border-box;margin-top:4px}
  .btn:hover{transform:translateY(-1px)}
  .danger{background:#4a1414;color:#d8a0a0;border-color:#6b2a2a} .danger:hover{background:#5c1a1a;border-color:#8b3a3a}
</style>

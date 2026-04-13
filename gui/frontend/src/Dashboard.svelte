<script>
  import { onMount, onDestroy } from 'svelte'
  import { DoStopTunnel, DoGetPeers, GetVirtualIP, GetNetworkID, GetUsername } from '../wailsjs/go/main/App.js'
  import { EventsOn } from '../wailsjs/runtime/runtime.js'
  import { lang, ui } from './lang.js'

  export let onDisconnect

  $: t = ui[$lang]

  let peers=[], logs=[], virtualIP='', networkID='', username='', status='connecting', logEl
  let cleanups = []

  onMount(async () => {
    virtualIP = await GetVirtualIP()
    networkID = await GetNetworkID()
    username = await GetUsername()
    peers = await DoGetPeers() || []
    cleanups.push(EventsOn('log', m => {
      logs = [...logs.slice(-200), m]
      setTimeout(() => { if(logEl) logEl.scrollTop = logEl.scrollHeight }, 10)
    }))
    cleanups.push(EventsOn('peers-updated', p => { peers = p || [] }))
    cleanups.push(EventsOn('status-changed', s => { status = s }))
  })
  onDestroy(() => cleanups.forEach(c => c()))

  async function disconnect() { await DoStopTunnel(); onDisconnect() }
</script>

<div class="panel fade-in">
  <div class="header">
    <h2 class="title">Pa<span class="lan">LAN</span>tir</h2>
    <div class="badge-status" class:on={status==='connected'} class:wait={status==='connecting'}>
      <div class="dot"></div>
      {status==='connected' ? t.connected : status==='connecting' ? t.connectingStatus : t.disconnected}
    </div>
  </div>

  <div class="info">
    <div class="info-cell"><span class="lbl">{t.user}</span><span class="val">{username}</span></div>
    <div class="info-cell"><span class="lbl">{t.virtualIP}</span><span class="val mono">{virtualIP}</span></div>
    <div class="info-cell"><span class="lbl">{t.netCode}</span><span class="val mono gold">{networkID}</span></div>
  </div>

  <div class="sec">
    <h3 class="sec-title">{t.peers} <span class="cnt">{peers.length}</span></h3>
    <div class="peer-list">
      {#if peers.length===0}
        <div class="empty"><div class="empty-dot"></div>{t.waitingPeers}</div>
      {/if}
      {#each peers as p, i}
        <div class="peer" style="animation-delay:{i*0.05}s">
          <div class="peer-l"><span class="pdot" class:on={p.ready}></span><span class="pname">{p.username}</span></div>
          <div class="peer-r">
            <span class="pip">{p.virtualIP}</span>
            <span class="mode" class:p2p={p.mode==='P2P'} class:relay={p.mode==='RELAY'}>{p.mode}</span>
          </div>
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

  <button class="btn btn-danger" on:click={disconnect}>{t.disconnect}</button>
</div>

<style>
  .panel {
    background: linear-gradient(170deg,#111a11,#0b120b);
    border:1px solid #2a3d2a; border-radius:clamp(8px,1.5vw,14px);
    padding:clamp(14px,3vh,22px) clamp(14px,3vw,22px);
    width:100%; max-width:min(520px,96vw); max-height:96vh;
    overflow-y:auto; box-shadow:0 24px 70px rgba(0,0,0,.5);
  }

  .header { display:flex; justify-content:space-between; align-items:center; margin-bottom:10px; }
  .title { font-family:'Cinzel',serif; font-size:clamp(1.2em,3.5vw,1.6em); color:#c9a84c; letter-spacing:2px; }
  .lan { color:#d4c88a; font-weight:900; }

  .badge-status {
    display:flex; align-items:center; gap:5px; font-size:.72em; color:#6b7b4a;
    padding:3px 9px; border:1px solid #2a3d2a; border-radius:10px; background:#0e150e;
  }
  .dot { width:7px; height:7px; border-radius:50%; background:#444; transition:all .3s; }
  .badge-status.on .dot { background:#5a8a3c; box-shadow:0 0 6px rgba(90,138,60,.6); }
  .badge-status.on { color:#8aaa6a; border-color:#2a4a2a; }
  .badge-status.wait .dot { background:#b8960c; animation:blink 1s infinite; }
  .badge-status.wait { color:#c9b06b; }
  @keyframes blink { 50%{opacity:.3} }

  .info {
    display:flex; gap:6px; background:rgba(100,160,60,.03); border:1px solid #1a2e1a;
    border-radius:7px; padding:8px 10px; margin-bottom:10px;
  }
  .info-cell { flex:1; display:flex; flex-direction:column; align-items:center; }
  .lbl { color:#4a5a3a; font-size:.65em; text-transform:uppercase; letter-spacing:1px; }
  .val { color:#c9b06b; font-weight:600; font-size:.85em; margin-top:1px; }
  .mono { font-family:'Consolas',monospace; letter-spacing:1px; }
  .gold { text-shadow:0 0 6px rgba(201,176,107,.2); }

  .sec { margin-bottom:10px; }
  .sec-title { color:#6b7b4a; font-size:.75em; text-transform:uppercase; letter-spacing:2px; font-family:'Cinzel',serif; margin:0 0 6px; }
  .cnt { color:#c9a84c; }

  .peer-list { display:flex; flex-direction:column; gap:4px; }
  .empty { color:#4a5a3a; font-style:italic; text-align:center; padding:14px; display:flex; flex-direction:column; align-items:center; gap:6px; }
  .empty-dot { width:10px; height:10px; border-radius:50%; background:#1a3040; box-shadow:0 0 6px rgba(30,70,100,.4); animation:blink 2s infinite; }

  .peer {
    display:flex; justify-content:space-between; align-items:center;
    background:rgba(100,160,60,.03); border:1px solid #1a2e1a;
    border-radius:5px; padding:7px 10px; transition:all .2s;
    animation: slideIn .4s ease-out backwards;
  }
  .peer:hover { background:rgba(100,160,60,.06); border-color:#2a3d2a; }
  @keyframes slideIn { from{opacity:0;transform:translateX(-12px)} to{opacity:1} }

  .peer-l { display:flex; align-items:center; gap:7px; }
  .pdot { width:7px; height:7px; border-radius:50%; background:#444; transition:all .3s; }
  .pdot.on { background:#5a8a3c; box-shadow:0 0 5px rgba(90,138,60,.5); }
  .pname { color:#c9b06b; font-weight:600; font-size:.9em; }
  .peer-r { display:flex; align-items:center; gap:7px; }
  .pip { color:#6b7b4a; font-family:monospace; font-size:.75em; }
  .mode { padding:2px 7px; border-radius:8px; font-size:.6em; font-weight:700; letter-spacing:1px; }
  .mode.p2p { background:#1a2e14; color:#8aaa6a; border:1px solid #2a4a2a; }
  .mode.relay { background:#2e2210; color:#c8a050; border:1px solid #4a3a1a; }

  .logbox {
    background:#060a06; border:1px solid #1a2e1a; border-radius:5px;
    padding:6px 8px; height:clamp(80px,18vh,160px); overflow-y:auto;
    font-family:'Consolas',monospace; font-size:.68em;
  }
  .logln { color:#5a7a4a; padding:1px 0; word-break:break-all; }
  .logln.dim { color:#2a3d2a; }

  .btn {
    width:100%; padding:10px; font-family:'Cinzel',serif; font-size:.88em;
    font-weight:600; border:1px solid; border-radius:6px; cursor:pointer;
    transition:all .25s; letter-spacing:1px;
  }
  .btn:hover { transform:translateY(-2px); }
  .btn-danger { background:#4a1414; color:#d8a0a0; border-color:#6b2a2a; }
  .btn-danger:hover { background:#5c1a1a; border-color:#8b3a3a; }

  .fade-in { animation:fadeIn .5s ease-out; }
  @keyframes fadeIn { from{opacity:0} to{opacity:1} }
</style>

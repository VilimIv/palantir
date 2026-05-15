<script>
  import { Quit } from '../wailsjs/runtime/runtime.js'
  import { lang, ui } from './lang.js'
  export let onLogin, onAbout, onSettings
  $: t = ui[$lang]

  let orbClicks=0, sauronOrb=false, sauronFull=false, timer=null
  function onOrbClick(){
    orbClicks++; clearTimeout(timer); timer=setTimeout(()=>{orbClicks=0},1200)
    if(orbClicks>=5){orbClicks=0;sauronOrb=false;sauronFull=true;setTimeout(()=>sauronFull=false,3000)}
    else if(orbClicks>=3&&!sauronOrb){sauronOrb=true;setTimeout(()=>sauronOrb=false,2500)}
  }
</script>

{#if sauronFull}
  <div class="sauron-overlay"><div class="eye-outer"><div class="eye-slit"></div></div></div>
{/if}

<div class="panel">
  <h1 class="title">Pa<span class="lan">LAN</span>tir</h1>
  <div class="orb-wrap">
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="orb" on:click={onOrbClick} class:shake={orbClicks>0} class:sauron-orb={sauronOrb}>
      {#if sauronOrb}<div class="mini-eye"><div class="mini-slit"></div></div>{:else}<div class="orb-shine"></div>{/if}
    </div>
  </div>
  <div class="buttons">
    <button class="btn primary" on:click={onLogin}>{t.loginBtn}</button>
    <button class="btn primary" on:click={onSettings}>{t.settingsBtn}</button>
    <button class="btn primary" on:click={onAbout}>{t.aboutBtn}</button>
    <button class="btn danger" on:click={()=>Quit()}>{t.exitBtn}</button>
  </div>
</div>

<style>
  .sauron-overlay{position:fixed;inset:0;z-index:9999;background:radial-gradient(ellipse,rgba(60,5,0,.95),rgba(3,0,0,.99) 70%);display:flex;justify-content:center;align-items:center;animation:sF 3s ease-in-out forwards}
  @keyframes sF{0%{opacity:0}12%{opacity:1}78%{opacity:1}100%{opacity:0}}
  .eye-outer{width:180px;height:80px;border-radius:50%;background:radial-gradient(ellipse,#ff4500,#bb1100 40%,#550000 75%,#1a0000);box-shadow:0 0 80px rgba(255,69,0,.8);position:relative;overflow:hidden;animation:eP .6s ease-in-out infinite}
  @keyframes eP{0%,100%{box-shadow:0 0 80px rgba(255,69,0,.8)}50%{box-shadow:0 0 110px rgba(255,100,0,1)}}
  .eye-slit{position:absolute;top:50%;left:50%;transform:translate(-50%,-50%);width:8px;height:78%;background:#000;border-radius:50%;animation:sP .5s ease-in-out infinite alternate}
  @keyframes sP{from{width:5px}to{width:14px}}
  .sauron-orb{background:radial-gradient(circle,#ff3300,#aa1500,#550000,#1a0000)!important;box-shadow:0 0 25px rgba(255,50,0,.7)!important;animation:sO .8s ease-in-out infinite!important}
  @keyframes sO{0%,100%{box-shadow:0 0 25px rgba(255,50,0,.7)}50%{box-shadow:0 0 35px rgba(255,80,0,.9)}}
  .mini-eye{position:absolute;top:50%;left:50%;transform:translate(-50%,-50%);width:70%;height:40%;border-radius:50%;background:radial-gradient(ellipse,#ff6600,#cc3300,transparent)}
  .mini-slit{position:absolute;top:50%;left:50%;transform:translate(-50%,-50%);width:3px;height:70%;background:#000;border-radius:50%;animation:mS .4s ease-in-out infinite alternate}
  @keyframes mS{from{width:2px}to{width:5px}}
  .panel{background:linear-gradient(170deg,#111a11,#0b120b 60%,#0e160e);border:1px solid #2a3d2a;border-radius:12px;padding:30px 26px;width:420px;text-align:center;box-shadow:0 0 50px rgba(60,100,40,.06),0 24px 70px rgba(0,0,0,.5)}
  .title{font-family:'Cinzel','Georgia',serif;font-size:2.5em;color:#c9a84c;letter-spacing:4px;text-shadow:0 0 25px rgba(201,168,76,.25)}
  .lan{color:#d4c88a;font-weight:900}
  .orb-wrap{display:flex;justify-content:center;margin:8px 0 24px}
  .orb{width:46px;height:46px;border-radius:50%;background:radial-gradient(circle at 35% 28%,#1a3040,#0a1520,#040810);box-shadow:0 0 22px rgba(30,70,100,.5);cursor:pointer;position:relative;overflow:hidden;animation:fl 5s ease-in-out infinite;transition:transform .12s}
  .orb:active{transform:scale(.9)} .orb.shake{animation:fl 5s ease-in-out infinite,wo .25s ease-in-out}
  .orb-shine{position:absolute;top:18%;left:22%;width:28%;height:28%;border-radius:50%;background:radial-gradient(circle,rgba(120,180,255,.45),transparent);filter:blur(3px)}
  @keyframes fl{0%,100%{transform:translateY(0)}50%{transform:translateY(-5px)}}
  @keyframes wo{25%{transform:rotate(-6deg) scale(1.12)}75%{transform:rotate(6deg) scale(1.12)}}
  .buttons{display:flex;flex-direction:column;gap:12px}
  .btn{padding:13px 20px;font-family:'Cinzel',serif;font-size:.98em;font-weight:600;letter-spacing:1.5px;border:1px solid;border-radius:7px;cursor:pointer;transition:all .25s}
  .btn:hover{transform:translateY(-2px);box-shadow:0 5px 18px rgba(0,0,0,.3)} .btn:active{transform:translateY(0)}
  .primary{background:linear-gradient(180deg,#1a2e1a,#142414);color:#c9b06b;border-color:#3a5a3a}
  .primary:hover{background:#243824;border-color:#5a8a3a;color:#d4c88a}
  .danger{background:linear-gradient(180deg,#4a1414,#3a0e0e);color:#d8a0a0;border-color:#6b2a2a}
  .danger:hover{background:#5c1a1a;border-color:#8b3a3a}
</style>

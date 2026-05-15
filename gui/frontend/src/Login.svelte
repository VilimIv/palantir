<script>
  import { DoLogin, DoRegister } from '../wailsjs/go/main/App.js'
  import { lang, ui } from './lang.js'
  export let onSuccess, onBack
  $: t = ui[$lang]
  let username='',password='',error='',success='',loading=false

  async function login(){error='';success='';loading=true;try{await DoLogin(username,password);success=t.loginSuccess;setTimeout(onSuccess,500)}catch(e){error=String(e)}loading=false}
  async function register(){error='';success='';loading=true;try{await DoRegister(username,password);success=t.registerSuccess;setTimeout(onSuccess,500)}catch(e){error=String(e)}loading=false}
</script>

<div class="panel">
  <h1 class="title">Pa<span class="lan">LAN</span>tir</h1>
  <p class="sub">{t.loginTitle}</p>
  <div class="form">
    <input type="text" placeholder={t.username} bind:value={username} on:keydown={e=>e.key==='Enter'&&login()} disabled={loading}/>
    <input type="password" placeholder={t.password} bind:value={password} on:keydown={e=>e.key==='Enter'&&login()} disabled={loading}/>
    {#if error}<div class="toast err">{error}</div>{/if}
    {#if success}<div class="toast ok">{success}</div>{/if}
    <div class="row">
      <button class="btn primary" on:click={login} disabled={loading||!username||!password}>{loading?'···':t.login}</button>
      <button class="btn primary" on:click={register} disabled={loading||!username||!password}>{loading?'···':t.register}</button>
    </div>
    <button class="btn back" on:click={onBack}>{t.back}</button>
  </div>
</div>

<style>
  .panel{background:linear-gradient(170deg,#111a11,#0b120b);border:1px solid #2a3d2a;border-radius:12px;padding:30px 26px;width:420px;text-align:center;box-shadow:0 24px 70px rgba(0,0,0,.5);overflow:hidden}
  .title{font-family:'Cinzel',serif;font-size:2.2em;color:#c9a84c;letter-spacing:3px} .lan{color:#d4c88a;font-weight:900}
  .sub{color:#6b7b4a;font-style:italic;margin:3px 0 20px;font-size:.9em}
  .form{display:flex;flex-direction:column;gap:10px}
  input{display:block;width:100%;padding:11px 14px;font-family:'Crimson Text',serif;font-size:1em;border:1px solid #2a3d2a;border-radius:6px;background:#152015;color:#d4c88a;outline:none;box-sizing:border-box}
  input:focus{border-color:#5a8a3a;box-shadow:0 0 10px rgba(90,138,58,.15)} input::placeholder{color:#4a5a3a} input:disabled{opacity:.5}
  .toast{padding:8px 12px;border-radius:5px;font-size:.85em}
  .toast.err{background:#2e0e0e;border:1px solid #8b2a2a;color:#e8a0a0;animation:shake .4s ease-out}
  .toast.ok{background:#1a2e14;border:1px solid #3a6b2a;color:#a0e8a0}
  @keyframes shake{0%{transform:translateX(-8px);opacity:0}25%{transform:translateX(6px)}50%{transform:translateX(-3px)}100%{transform:translateX(0);opacity:1}}
  .row{display:flex;gap:8px}
  .btn{flex:1;padding:11px 14px;font-family:'Cinzel',serif;font-size:.9em;font-weight:600;border:1px solid;border-radius:6px;cursor:pointer;transition:all .25s;box-sizing:border-box}
  .btn:hover:not(:disabled){transform:translateY(-2px)} .btn:disabled{opacity:.4;cursor:not-allowed}
  .primary{background:linear-gradient(180deg,#1a2e1a,#142414);color:#c9b06b;border-color:#3a5a3a}
  .primary:hover:not(:disabled){background:#243824;border-color:#5a8a3a}
  .back{flex:none;background:transparent;color:#6b7b4a;border-color:#2a3d2a;margin-top:3px} .back:hover{color:#c9b06b;border-color:#3a5a3a}
</style>

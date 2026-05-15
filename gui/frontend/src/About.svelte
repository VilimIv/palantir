<script>
  import { lang, ui } from './lang.js'
  import { encyclopedia } from './encyclopedia.js'
  import { GetSarumanGif } from '../wailsjs/go/main/App.js'
  import { onMount } from 'svelte'
  export let onBack

  $: t = ui[$lang]
  $: enc = encyclopedia[$lang]
  $: categories = enc ? Object.keys(enc) : []

  let selectedCatIndex = -1, selectedTopicIndex = -1, searchQuery = ''
  let prevLang = null, sarumanGif = ''

  onMount(async () => {
    sarumanGif = await GetSarumanGif()
  })

  $: { if (prevLang !== null && prevLang !== $lang) { /* indexes stay */ } prevLang = $lang }

  $: selectedTopic = (selectedCatIndex >= 0 && selectedTopicIndex >= 0 && enc)
    ? enc[categories[selectedCatIndex]]?.[selectedTopicIndex] ?? null : null

  function selectTopic(ci, ti) { selectedCatIndex=ci; selectedTopicIndex=ti }
  function clearTopic() { selectedCatIndex=-1; selectedTopicIndex=-1 }

  $: filteredTopics = searchQuery && enc
    ? Object.entries(enc).flatMap(([cat, topics], ci) =>
        topics.filter(tp => tp.title.toLowerCase().includes(searchQuery.toLowerCase()))
              .map(tp => ({...tp, category:cat, catIdx:ci, topicIdx:topics.indexOf(tp)}))) : []

  // Zamijeni image placeholder s pravim GIF-om
  $: resolvedTopic = selectedTopic ? {
    ...selectedTopic,
    image: selectedTopic.image === 'sarumanGif' ? sarumanGif : null
  } : null
</script>

<div class="panel">
  <h2 class="title">Pa<span class="lan">LAN</span>tir</h2>
  <p class="sub">{t.encyclopediaTitle}</p>
  <input class="search" type="text" placeholder={t.searchPlaceholder} bind:value={searchQuery}/>
  <div class="content">
    {#if searchQuery && filteredTopics.length > 0}
      {#each filteredTopics as topic}
        <button class="tbtn" on:click={()=>{selectTopic(topic.catIdx,topic.topicIdx);searchQuery=''}}>
          <span class="tn">{topic.title}</span><span class="tc">{topic.category}</span>
        </button>
      {/each}
    {:else if resolvedTopic}
      <div class="detail">
        <button class="link" on:click={clearTopic}>{t.back}</button>
        <h3 class="dt">{resolvedTopic.title}</h3>
        <div class="desc">{resolvedTopic.description}</div>
        {#if resolvedTopic.diagram}<pre class="diag">{resolvedTopic.diagram}</pre>{/if}
        {#if resolvedTopic.image}<img src={resolvedTopic.image} alt="" class="timg"/>{/if}
        {#if resolvedTopic.inPalantir}<div class="pbox"><strong>{t.inPalantir}</strong><p>{resolvedTopic.inPalantir}</p></div>{/if}
      </div>
    {:else}
      {#each categories as cat, ci}
        <div class="cat">
          <h3 class="cn">{cat}</h3>
          <div class="chips">{#each enc[cat] as topic, ti}<button class="chip" on:click={()=>selectTopic(ci,ti)}>{topic.title}</button>{/each}</div>
        </div>
      {/each}
    {/if}
  </div>
  <button class="bbtn" on:click={onBack}>{t.back}</button>
</div>

<style>
  .panel{background:linear-gradient(170deg,#111a11,#0b120b);border:1px solid #2a3d2a;border-radius:12px;padding:14px 18px;width:420px;height:600px;display:flex;flex-direction:column;text-align:center;box-shadow:0 24px 70px rgba(0,0,0,.5);overflow:hidden}
  .title{font-family:'Cinzel',serif;font-size:1.4em;color:#c9a84c;letter-spacing:2px} .lan{color:#d4c88a;font-weight:900}
  .sub{color:#6b7b4a;font-size:.78em;margin:1px 0 8px}
  .search{display:block;width:100%;box-sizing:border-box;padding:7px 10px;font-family:'Crimson Text',serif;font-size:.88em;border:1px solid #2a3d2a;border-radius:5px;background:#152015;color:#d4c88a;outline:none;margin-bottom:6px}
  .search:focus{border-color:#5a8a3a} .search::placeholder{color:#4a5a3a}
  .content{flex:1;overflow-y:auto;overflow-x:hidden;text-align:left;min-height:0}
  .cat{margin-bottom:10px}
  .cn{color:#b0cc7a;font-family:'Cinzel',serif;font-size:.7em;letter-spacing:2px;text-transform:uppercase;margin:0 0 5px;border-bottom:1px solid #2a3d2a;padding-bottom:3px}
  .chips{display:flex;flex-wrap:wrap;gap:3px}
  .chip{background:#1a2e1a;color:#c9b06b;border:1px solid #2a4a2a;border-radius:10px;padding:2px 9px;font-size:.72em;cursor:pointer;transition:all .2s;font-family:'Crimson Text',serif}
  .chip:hover{background:#243824;border-color:#5a8a3a}
  .tbtn{display:flex;justify-content:space-between;align-items:center;width:100%;background:rgba(100,160,60,.03);border:1px solid #1a2e1a;border-radius:5px;padding:6px 9px;cursor:pointer;transition:all .2s;text-align:left;margin-bottom:3px;box-sizing:border-box}
  .tbtn:hover{background:rgba(100,160,60,.08)} .tn{color:#c9b06b;font-weight:600;font-size:.84em} .tc{color:#4a5a3a;font-size:.6em}
  .detail{padding:0}
  .link{background:none;border:none;color:#5a8a3a;cursor:pointer;font-size:.78em;padding:0;margin-bottom:3px} .link:hover{text-decoration:underline}
  .dt{color:#c9a84c;font-family:'Cinzel',serif;font-size:.95em;margin:3px 0 6px}
  .desc{color:#d0d8b0;line-height:1.6;font-size:.82em;white-space:pre-line;word-wrap:break-word}
  .diag{background:#080e08;border:1px solid #1a2e1a;border-radius:4px;padding:8px;margin:8px 0;font-family:'Consolas',monospace;font-size:.58em;color:#a0c080;overflow-x:auto;line-height:1.35;white-space:pre;user-select:text;max-width:100%}
  .timg{width:100%;border-radius:5px;margin:10px 0;border:1px solid #1a2e1a}
  .pbox{background:rgba(90,138,58,.12);border-left:3px solid #6aaa4a;padding:8px 12px;border-radius:0 6px 6px 0;margin-top:10px;font-size:.82em}
  .pbox strong{color:#a0cc6a;font-size:.95em} .pbox p{margin:4px 0 0;color:#b8c898;line-height:1.5}
  .bbtn{margin-top:6px;width:100%;padding:7px;background:transparent;color:#6b7b4a;border:1px solid #2a3d2a;border-radius:5px;cursor:pointer;font-family:'Cinzel',serif;font-size:.8em;transition:all .2s;letter-spacing:1px;flex-shrink:0;box-sizing:border-box}
  .bbtn:hover{color:#c9b06b;border-color:#3a5a3a}
</style>

<script>
  import { lang, ui } from './lang.js'
  import { encyclopedia } from './encyclopedia.js'
  export let onBack

  $: t = ui[$lang]
  $: enc = encyclopedia[$lang]
  $: categories = enc ? Object.keys(enc) : []

  let selectedCatIndex = -1, selectedTopicIndex = -1, searchQuery = ''
  let prevLang = null
  $: { if (prevLang !== null && prevLang !== $lang) { /* indexes stay, topic auto-updates */ } prevLang = $lang }

  $: selectedTopic = (selectedCatIndex >= 0 && selectedTopicIndex >= 0 && enc)
    ? enc[categories[selectedCatIndex]]?.[selectedTopicIndex] ?? null : null

  function selectTopic(ci, ti) { selectedCatIndex=ci; selectedTopicIndex=ti }
  function clearTopic() { selectedCatIndex=-1; selectedTopicIndex=-1 }

  $: filteredTopics = searchQuery && enc
    ? Object.entries(enc).flatMap(([cat, topics], ci) =>
        topics.filter(tp => tp.title.toLowerCase().includes(searchQuery.toLowerCase()))
              .map(tp => ({...tp, category:cat, catIdx:ci, topicIdx:topics.indexOf(tp)}))) : []
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
    {:else if selectedTopic}
      <div class="detail">
        <button class="link" on:click={clearTopic}>{t.back}</button>
        <h3 class="dt">{selectedTopic.title}</h3>
        <div class="desc">{selectedTopic.description}</div>
        {#if selectedTopic.diagram}<pre class="diag">{selectedTopic.diagram}</pre>{/if}
        {#if selectedTopic.image}<img src={selectedTopic.image} alt="" class="timg"/>{/if}
        {#if selectedTopic.inPalantir}<div class="pbox"><strong>{t.inPalantir}</strong><p>{selectedTopic.inPalantir}</p></div>{/if}
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
  .panel{
    background:linear-gradient(170deg,#111a11,#0b120b);
    border:1px solid #2a3d2a;border-radius:12px;
    padding:14px 18px;width:420px;height:600px;
    display:flex;flex-direction:column;text-align:center;
    box-shadow:0 24px 70px rgba(0,0,0,.5);
    overflow:hidden;
  }
  .title{font-family:'Cinzel',serif;font-size:1.4em;color:#c9a84c;letter-spacing:2px}
  .lan{color:#d4c88a;font-weight:900}
  .sub{color:#6b7b4a;font-size:.78em;margin:1px 0 8px}

  .search{
    display:block;width:100%;box-sizing:border-box;
    padding:7px 10px;font-family:'Crimson Text',serif;font-size:.88em;
    border:1px solid #2a3d2a;border-radius:5px;background:#152015;
    color:#d4c88a;outline:none;margin-bottom:6px;
  }
  .search:focus{border-color:#5a8a3a}
  .search::placeholder{color:#4a5a3a}

  .content{flex:1;overflow-y:auto;overflow-x:hidden;text-align:left;min-height:0}

  .cat{margin-bottom:10px}
  .cn{color:#8aaa5a;font-family:'Cinzel',serif;font-size:.68em;letter-spacing:2px;text-transform:uppercase;margin:0 0 4px;border-bottom:1px solid #1a2e1a;padding-bottom:2px}
  .chips{display:flex;flex-wrap:wrap;gap:3px}
  .chip{background:#1a2e1a;color:#c9b06b;border:1px solid #2a4a2a;border-radius:10px;padding:2px 9px;font-size:.72em;cursor:pointer;transition:all .2s;font-family:'Crimson Text',serif}
  .chip:hover{background:#243824;border-color:#5a8a3a}

  .tbtn{display:flex;justify-content:space-between;align-items:center;width:100%;background:rgba(100,160,60,.03);border:1px solid #1a2e1a;border-radius:5px;padding:6px 9px;cursor:pointer;transition:all .2s;text-align:left;margin-bottom:3px;box-sizing:border-box}
  .tbtn:hover{background:rgba(100,160,60,.08)}
  .tn{color:#c9b06b;font-weight:600;font-size:.84em}
  .tc{color:#4a5a3a;font-size:.6em}

  .detail{padding:0}
  .link{background:none;border:none;color:#5a8a3a;cursor:pointer;font-size:.78em;padding:0;margin-bottom:3px}
  .link:hover{text-decoration:underline}
  .dt{color:#c9a84c;font-family:'Cinzel',serif;font-size:.95em;margin:3px 0 6px}
  .desc{color:#a0aa80;line-height:1.5;font-size:.8em;white-space:pre-line;word-wrap:break-word}

  .diag{
    background:#060a06;border:1px solid #1a2e1a;border-radius:4px;
    padding:8px;margin:8px 0;font-family:'Consolas',monospace;
    font-size:.55em;color:#6b8a4a;overflow-x:auto;line-height:1.3;
    white-space:pre;user-select:text;max-width:100%;
  }

  .timg{width:100%;border-radius:5px;margin:10px 0;border:1px solid #1a2e1a}

  .pbox{background:rgba(90,138,58,.08);border-left:3px solid #5a8a3a;padding:6px 10px;border-radius:0 5px 5px 0;margin-top:8px;font-size:.78em}
  .pbox strong{color:#8aaa5a}
  .pbox p{margin:2px 0 0;color:#8a9a6a}

  .bbtn{margin-top:6px;width:100%;padding:7px;background:transparent;color:#6b7b4a;border:1px solid #2a3d2a;border-radius:5px;cursor:pointer;font-family:'Cinzel',serif;font-size:.8em;transition:all .2s;letter-spacing:1px;flex-shrink:0;box-sizing:border-box}
  .bbtn:hover{color:#c9b06b;border-color:#3a5a3a}
</style>

package gateway

const adminHTML = `<!doctype html><html lang="zh"><head><meta charset="utf-8"><title>Model Gateway</title>
<style>body{font-family:sans-serif;max-width:960px;margin:2em auto;padding:0 1em}table{border-collapse:collapse;width:100%}
td,th{border:1px solid #ccc;padding:6px;font-size:14px}input,select{margin:2px;padding:4px}.off{color:#999}</style></head><body>
<h2>API Key 池</h2>
<form id="f"><input name="name" placeholder="名称"><input name="base_url" placeholder="Base URL (如 https://api.openai.com/v1)" size="40">
<select name="protocol"><option>openai</option><option>anthropic</option></select>
<input name="model" placeholder="上游模型名(可空)"><input name="api_key" placeholder="API Key" size="30"><button>添加</button></form>
<table><thead><tr><th>名称</th><th>Base URL</th><th>协议</th><th>模型</th><th>Key</th><th>状态</th><th>最近错误</th><th>操作</th></tr></thead><tbody id="t"></tbody></table>
<script>
const esc=s=>String(s??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
async function load(){const r=await fetch('/admin/api/keys');const l=await r.json();
document.getElementById('t').innerHTML=l.map(e=>'<tr class="'+(e.enabled?'':'off')+'"><td>'+esc(e.name)+'</td><td>'+esc(e.base_url)+'</td><td>'+esc(e.protocol)+'</td><td>'+esc(e.model)+'</td><td>'+esc(e.api_key)+'</td><td>'+(e.enabled?'启用':'禁用')+'</td><td>'+esc(e.last_error)+'</td><td><button data-a="tg" data-id="'+esc(e.id)+'" data-en="'+(!e.enabled)+'">'+(e.enabled?'禁用':'启用')+'</button> <button data-a="del" data-id="'+esc(e.id)+'">删除</button></td></tr>').join('')}
document.getElementById('t').onclick=async ev=>{const b=ev.target;if(!b.dataset.a)return;
if(b.dataset.a==='del'){if(!confirm('确认删除?'))return;await fetch('/admin/api/keys/'+b.dataset.id,{method:'DELETE'})}
else await fetch('/admin/api/keys/'+b.dataset.id+'/enabled',{method:'POST',body:JSON.stringify({enabled:b.dataset.en==='true'})});load()};
document.getElementById('f').onsubmit=async ev=>{ev.preventDefault();const d=Object.fromEntries(new FormData(ev.target));
const r=await fetch('/admin/api/keys',{method:'POST',body:JSON.stringify(d)});if(!r.ok){alert((await r.json()).error.message);return}ev.target.reset();load()};
load();setInterval(load,5000);
</script></body></html>`

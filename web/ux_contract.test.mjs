import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import vm from 'node:vm';
const html=readFileSync(new URL('./index.html', import.meta.url),'utf8');
const beforeScript=html.slice(0,html.indexOf('<script>'));
const script=html.slice(html.indexOf('<script>')+'<script>'.length,html.lastIndexOf('</script>'));
const matchCount=(s,r)=>[...s.matchAll(r)].length;

assert.equal(matchCount(beforeScript,/<main\b/gi),1,'exactly one main landmark');
assert.equal(matchCount(beforeScript,/<h1\b/gi),1,'one page title');
assert.equal(matchCount(beforeScript,/<nav\b/gi),1,'one primary navigation');
assert.equal(matchCount(beforeScript,/<footer\b/gi),1,'research-only footer');
assert.match(beforeScript,/<a class="skip-link" href="#main-content">/);
assert.match(beforeScript,/<html lang="en">/);
assert.match(beforeScript,/<form class="search-form" id="q">/);
assert.match(beforeScript,/<input id="query" required maxlength="512"/);
assert.match(beforeScript,/<details class="advanced" id="advanced-settings"><summary>/);
assert.ok(beforeScript.indexOf('id="query"')<beforeScript.indexOf('id="advanced-settings"'),'primary search precedes advanced options');
assert.ok(!beforeScript.includes('<details class="advanced" id="advanced-settings" open'),'advanced controls initially collapsed');
assert.match(beforeScript,/id="results"[^>]*aria-labelledby="results-heading"/);
for(const id of ['concepts','observations','sweep','technical-evidence']) assert.ok(beforeScript.includes('id="'+id+'"'));
assert.match(beforeScript,/id="mode" role="status"/);
assert.match(beforeScript,/id="status" role="status" aria-live="polite"/);
assert.doesNotMatch(beforeScript,/max="3 value="3"/);
assert.match(beforeScript,/no torrent payload is downloaded/i);
assert.match(beforeScript,/not CPE assessments or license determinations/);
assert.match(beforeScript,/\/api\/v1\/capabilities/);
assert.match(beforeScript,/\/api\/v1\/concepts/);
assert.match(beforeScript,/class="hero-signal" aria-hidden="true"/);
assert.match(beforeScript,/prefers-reduced-motion:reduce/);
assert.match(beforeScript,/forced-colors:active/);
assert.match(beforeScript,/@media\(max-width:520px\)/);
assert.match(beforeScript,/@media \(prefers-color-scheme:dark\)/);
assert.match(beforeScript,/:root\[data-theme="dark"\]/);
assert.match(beforeScript,/:focus-visible/);

const idMatches=[...beforeScript.matchAll(/\bid="([^"]+)"/g)].map(m=>m[1]);
assert.equal(new Set(idMatches).size,idMatches.length,'IDs must be unique');
for(const id of ['query','terms','languages','media','minseeders','minleechers','adult','sort',
 'group','concept-filter','sweep-languages','sweep-local-budget','sweep-external-budget']){
 assert.ok(idMatches.includes(id),'missing source contract id: '+id);
 assert.ok(beforeScript.includes('for="'+id+'"'),'missing label for: '+id);
}
for(const anchor of ['#search','#results','#concepts','#sweep'])
 assert.ok(beforeScript.includes('href="'+anchor+'"'),'missing internal task navigation: '+anchor);

const radios=[...beforeScript.matchAll(/<input class="theme-choice" type="radio" name="bce-theme" value="([^"]+)"[^>]*aria-label="([^"]+)"/g)];
assert.deepEqual(radios.map(x=>x[1]),['system','light','dark'],'three native theme radios');
assert.deepEqual(radios.map(x=>x[2]),['System theme','Light theme','Dark theme']);
assert.match(beforeScript,/<fieldset class="theme-picker"><legend>Theme<\/legend>/);

for(const phrase of ['Fedora','A Princess of Mars','Ubuntu'])
 assert.ok(beforeScript.includes('data-query="'+phrase+'"'));
assert.match(script,/button\.addEventListener\('click',\(\)=>/);
assert.match(script,/query\.focus\(\)/);
assert.match(script,/localStorage\.getItem\('bce-ui-theme'\)/);
assert.match(script,/localStorage\.setItem\('bce-ui-theme'/);
assert.match(script,/if\(value==='system'\)document\.documentElement\.removeAttribute\('data-theme'\)/);
assert.match(script,/document\.querySelectorAll\('input\[name="bce-theme"\]'\)/);
new vm.Script(script,{filename:'web/index.html inline application script'});

const contrast=(a,b)=>{
 const values=[a,b].map(hex=>{
  const channels=[1,3,5].map(i=>parseInt(hex.slice(i,i+2),16)/255)
    .map(v=>v<=.04045?v/12.92:((v+.055)/1.055)**2.4);
  return .2126*channels[0]+.7152*channels[1]+.0722*channels[2];
 });
 return (Math.max(...values)+.05)/(Math.min(...values)+.05);
};
for(const [fg,bg,label] of [
 ['#183039','#f3f6f5','light main'],['#4e6670','#fffefa','light muted'],
 ['#edf7f6','#091820','dark main'],['#bed3d5','#122a34','dark muted']
]){
 assert.ok(contrast(fg,bg)>=4.5,label+' contrast below 4.5');
}
console.log('PASS: task-first workbench landmarks, unique IDs, labels, themes, contrast and inline JS syntax');

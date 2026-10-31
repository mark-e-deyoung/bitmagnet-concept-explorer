import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';

const html = readFileSync(new URL('./index.html', import.meta.url), 'utf8');
assert.ok(html.includes('id="sweep-external-budget" type="number" min="1" max="3" value="3"'), 'multilingual sweep request budget has valid bounds and default');
for (const id of [
  'research-evidence-title', 'research-evidence-section',
  'research-evidence-filter', 'research-evidence-load',
  'research-evidence-status', 'research-evidence-records',
]) {
  assert.ok(html.includes(`id="${id}"`), `research UI control exists: ${id}`);
}
const begin = html.indexOf('let researchEvidenceToken=0;');
const end = html.indexOf("\n$('sweep-limits').addEventListener('click',async()=>{", begin);
assert.ok(begin > 0 && end > begin, 'actual evidence inspection UI handler is present');
const handlerSource = html.slice(begin, end);
assert.ok(!handlerSource.includes('innerHTML'), 'untrusted source data is never inserted as HTML');

function element() {
  return {
    children: [], listeners: {}, value: '', disabled: false, textContent: '',
    append(...items) { this.children.push(...items); },
    replaceChildren() { this.children = []; this.textContent = ''; },
    addEventListener(name, handler) { this.listeners[name] = handler; },
  };
}
function harness(fetch) {
  const ids = ['research-evidence-section', 'research-evidence-filter', 'research-evidence-load', 'research-evidence-status', 'research-evidence-records'];
  const elements = new Map(ids.map(id => [id, element()]));
  elements.get('research-evidence-section').value = 'claims';
  elements.get('research-evidence-filter').value = 'Barsoom';
  const document = {
    getElementById(id) { return elements.get(id); },
    createElement() { return element(); },
  };
  const context = { document, fetch, URLSearchParams, console };
  vm.runInNewContext(`const $=id=>document.getElementById(id); ${handlerSource}`, context);
  return { elements, click: elements.get('research-evidence-load').listeners.click };
}

const safe = harness(async url => {
  assert.equal(url, '/api/v1/research-evidence?section=claims&q=Barsoom');
  return {
    ok: true, json: async () => ({
      authority_status: 'research-only-no-accepted-cpe-facts',
      digest: 'sha256:' + 'a'.repeat(64),
      items: [{
        id: 'claim:synthetic', predicate: 'catalog_subject_mentions',
        object: '<script>not executable</script>', status: 'source_report_not_independently_qualified',
        source_url: 'https://www.gutenberg.org/ebooks/62',
      }],
    }),
  };
});
await safe.click();
assert.match(safe.elements.get('research-evidence-status').textContent, /Research only/);
const article = safe.elements.get('research-evidence-records').children[0];
assert.ok(article.children.some(item => item.textContent?.includes('<script>not executable</script>')),
  'untrusted source text remains inert text');
const link = article.children.find(item => item.href);
assert.equal(link.rel, 'noopener noreferrer');
assert.equal(link.href, 'https://www.gutenberg.org/ebooks/62');

const fail = harness(async () => ({ ok: false, json: async () => ({error:'not configured'}) }));
await fail.click();
assert.match(fail.elements.get('research-evidence-status').textContent, /no absence conclusion/);
assert.equal(fail.elements.get('research-evidence-records').children.length, 0);

const pending = [];
const stale = harness(() => new Promise(resolve => pending.push(resolve)));
const first = stale.click();
const second = stale.click();
pending[1]({ ok:true, json:async()=>({
  authority_status:'research-only-no-accepted-cpe-facts', digest:'sha256:current',
  items:[{id:'second'}],
})});
await second;
pending[0]({ ok:true, json:async()=>({
  authority_status:'research-only-no-accepted-cpe-facts', digest:'sha256:stale',
  items:[{id:'first'}],
})});
await first;
assert.match(stale.elements.get('research-evidence-status').textContent, /current/);
assert.ok(stale.elements.get('research-evidence-records').children[0].children[0].textContent.includes('second'));
console.log('Public research evidence UI source, safe rendering, failure and stale-response qualification passed');

import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import vm from 'node:vm';

// Exercise the shipped form handler. This is a DOM-independent request-contract
// regression, not a replacement for browser/assistive-technology qualification.
const html = readFileSync(new URL('./index.html', import.meta.url), 'utf8');
function extract(startMarker, endMarker) {
  const start = html.indexOf(startMarker);
  const end = html.indexOf(endMarker, start);
  assert.ok(start >= 0 && end > start, `production source exists: ${startMarker}`);
  return html.slice(start, end);
}
const handler = extract("$('feedback-preview-form').addEventListener('submit',async event=>{", '\ncapabilities();loadCatalog();');
const queryBuilder = extract('function buildQueryRequest()', '\nfunction clearQueryEvidence(');
const statusHelpers = extract('function setStatus(', '\nfunction displayItems(');
const panelHelpers = extract('function finishPanelRequest(', "$('media-research-form').addEventListener");
const elements = new Map();
const formFields = {
  q: ['query', 'terms', 'languages', 'media', 'minseeders', 'minleechers', 'adult', 'sort', 'group', 'web'],
  'media-research-form': ['lookup-title', 'lookup-provider', 'lookup-kind', 'lookup-year', 'lookup-external-id', 'lookup-genres', 'lookup-related', 'lookup-consent'],
  'feedback-preview-form': ['feedback-candidate', 'feedback-language', 'feedback-kind'],
};
const document = {activeElement: null, querySelectorAll: () => []};
const $ = id => {
  if (!elements.has(id)) elements.set(id, {
    id, value: '', checked: false, disabled: false, textContent: '', children: [], attributes: new Map(), dataset: {}, listeners: {},
    setAttribute(name, value) { this.attributes.set(name, String(value)); },
    getAttribute(name) { return this.attributes.get(name) ?? null; },
    removeAttribute(name) { this.attributes.delete(name); },
    replaceChildren() { this.children = []; this.textContent = ''; },
    querySelectorAll(selector) {
      assert.equal(selector, '[aria-invalid="true"]', 'fixture supports the actual invalid-field selector');
      return (formFields[this.id] || []).map($).filter(field => field.getAttribute('aria-invalid') === 'true');
    },
    closest() { return null; },
    focus() { document.activeElement = this; },
    reportValidity() { return true; },
    addEventListener(name, listener) { this.listeners[name] = listener; },
  });
  return elements.get(id);
};
for (const [id, value] of Object.entries({
  query: 'Synthetic animation', terms: '  Entwurf  \n\n เทศกาล \n',
  languages: 'en, de', media: 'movie', adult: 'no', minseeders: '2', minleechers: '0',
  sort: 'seeders_desc', group: 'track', 'feedback-kind': 'lexicalization_candidate',
  'feedback-candidate': '  Synthetic candidate  ', 'feedback-language': ' de ',
})) $(id).value = value;
$('use-concept').checked = true;
$('web').checked = true;
const requests = [];
const context = {
  $, document, AbortController, setTimeout, clearTimeout,
  fetch: async (url, options) => {
    requests.push({url, body: JSON.parse(options.body)});
    return {ok: true, json: async () => ({state: 'local_review_export_only', observation_count: 1})};
  },
};
vm.runInNewContext(`let feedbackPreviewToken=0,lookupToken=0,activeFeedbackController=null,activeLookupController=null,focusConcept={id:'fixture.media'};\n${statusHelpers}\n${panelHelpers}\n${queryBuilder}\n${handler}`, context);
assert.equal(requests.length, 0, 'registering the human feedback workflow never sends a request');
const expectedScope = {
  query: 'Synthetic animation',
  expansions: [
    {term: 'Entwurf', provenance: 'manual-unverified'},
    {term: 'เทศกาล', provenance: 'manual-unverified'},
  ],
  filters: {languages: ['en', 'de'], content_types: ['movie'], adult_metadata: 'no', min_seeders: 2, min_leechers: 0},
  sort: 'seeders_desc', group_by: 'track', include_web: false, concept_ids: ['fixture.media'],
};
await $('feedback-preview-form').listeners.submit({preventDefault() {}});
assert.equal(requests.length, 1, 'only explicit feedback submission creates the local preview request');
assert.equal(requests[0].url, '/api/v1/discovery/feedback/preview');
assert.deepEqual(requests[0].body, {
  query: expectedScope, proposal_kind: 'lexicalization_candidate',
  candidate_label: 'Synthetic candidate', language: 'de',
}, 'feedback must preserve the selected query scope and explicitly exclude external web retrieval');
assert.equal($('web').checked, true, 'building a local preview does not mutate another workflow consent checkbox');
assert.equal($('feedback-submit').disabled, false, 'a completed feedback preview remains repeatable');

// A selected catalog node is context, not permission to expand a query.
$('use-concept').checked = false;
$('minseeders').value = '';
$('minleechers').value = '';
await $('feedback-preview-form').listeners.submit({preventDefault() {}});
assert.deepEqual(requests[1].body.query.concept_ids, [], 'unchecked concept expansion stays excluded');
assert.equal(Object.hasOwn(requests[1].body.query.filters, 'min_seeders'), false, 'unset seeders is absent, not a forced zero');
assert.equal(Object.hasOwn(requests[1].body.query.filters, 'min_leechers'), false, 'unset leechers is absent, not a forced zero');
assert.equal(requests[1].body.query.include_web, false);

$('query').value = '   ';
await $('feedback-preview-form').listeners.submit({preventDefault() {}});
assert.equal(requests.length, 2, 'empty query fails locally before any retrieval');
assert.equal($('feedback-status').attributes.get('data-ui-state'), 'blocked');
assert.equal($('query').getAttribute('aria-invalid'), 'true', 'empty query is associated with its explanatory error');
assert.equal(document.activeElement, $('query'), 'empty query moves focus to the field that needs correction');

const consentListeners = extract('// Consent belongs to the reviewed scope', '// Theme is an optional presentation preference');
vm.runInNewContext(consentListeners, context);
const armSearchConsent = () => { for (const id of ['web', 'sweep-web', 'sweep-consent']) $(id).checked = true; };
const assertSearchConsentCleared = label => {
  for (const id of ['web', 'sweep-web', 'sweep-consent']) assert.equal($(id).checked, false, `${label} clears ${id}`);
};
const requestCountBeforeEdits = requests.length;
for (const eventName of ['input', 'change']) {
  for (const id of ['query', 'terms', 'languages', 'media', 'minseeders', 'minleechers', 'adult', 'sort', 'group']) {
    armSearchConsent();
    $('q').listeners[eventName]({target: $(id)});
    assertSearchConsentCleared(`${id} ${eventName}`);
  }
  armSearchConsent();
  $('q').listeners[eventName]({target: $('web')});
  assert.equal($('web').checked, true, 'explicit web opt-in is not cleared by its own event');
  for (const id of ['lookup-title', 'lookup-provider', 'lookup-kind', 'lookup-year', 'lookup-external-id', 'lookup-genres', 'lookup-related']) {
    $('lookup-consent').checked = true;
    $('media-research-form').listeners[eventName]({target: $(id)});
    assert.equal($('lookup-consent').checked, false, `${id} ${eventName} requires fresh external source consent`);
  }
  $('lookup-consent').checked = true;
  $('media-research-form').listeners[eventName]({target: $('lookup-consent')});
  assert.equal($('lookup-consent').checked, true, 'explicit source consent is not cleared by its own event');
}
armSearchConsent();
$('use-concept').listeners.change();
assertSearchConsentCleared('concept expansion opt-in change');
for (const id of ['sweep-languages', 'sweep-local-budget', 'sweep-external-budget']) {
  armSearchConsent();
  $(id).listeners.input();
  assertSearchConsentCleared(`${id} edit`);
}
assert.equal(requests.length, requestCountBeforeEdits, 'editing scope and consent never starts retrieval or automatically retries');

$('query').value = 'Synthetic animation';
$('q').reportValidity = () => false;
await $('feedback-preview-form').listeners.submit({preventDefault() {}});
assert.equal(requests.length, requestCountBeforeEdits, 'invalid native query controls block feedback before retrieval');
$('q').reportValidity = () => true;

for (const lateOutcome of ['success', 'failure']) {
  let resolve, reject, signal;
  const pending = new Promise((yes, no) => { resolve = yes; reject = no; });
  context.fetch = (_url, options) => { signal = options.signal; return pending; };
  const preview = $('feedback-preview-form').listeners.submit({preventDefault() {}});
  assert.equal($('feedback-submit').disabled, true);
  assert.equal($('feedback-cancel').disabled, false);
  assert.equal($('feedback-output').getAttribute('aria-busy'), 'true');
  document.activeElement = $('feedback-cancel');
  $('feedback-cancel').listeners.click();
  assert.equal(signal.aborted, true, 'feedback cancellation aborts its pending fetch');
  assert.equal($('feedback-submit').disabled, false, 'canceling feedback immediately restores preview');
  assert.equal($('feedback-cancel').disabled, true);
  assert.equal($('feedback-output').getAttribute('aria-busy'), 'false');
  assert.equal(document.activeElement, $('feedback-submit'), 'focus returns to a usable action after feedback cancellation');
  const canceledStatus = $('feedback-status').textContent;
  if (lateOutcome === 'success') resolve({ok: true, json: async () => ({state: 'local_review_export_only', observation_count: 99})});
  else reject(new Error('late synthetic failure'));
  await preview;
  assert.equal($('feedback-status').textContent, canceledStatus, `late ${lateOutcome} cannot replace cancellation`);
  assert.equal($('feedback-output').textContent, '', `late ${lateOutcome} cannot restore stale feedback evidence`);
  assert.equal($('feedback-submit').disabled, false);
}

const lookupHandler = extract("$('media-research-form').addEventListener('submit',async event=>{", "$('feedback-preview-form').addEventListener('submit',async event=>{");
vm.runInNewContext(lookupHandler, context);
for (const [id, value] of Object.entries({
  'lookup-provider': 'jellyfin', 'lookup-kind': 'movie', 'lookup-title': 'Conflicting synthetic title',
  'lookup-genres': 'Animation', 'lookup-year': '', 'lookup-external-id': '',
})) $(id).value = value;
$('lookup-related').checked = false;
$('lookup-consent').checked = false;
$('lookup-genres').setAttribute('aria-describedby', 'lookup-genre-hint');
const lookupRequests = [];
context.fetch = async (url, options) => {
  lookupRequests.push({url, body: JSON.parse(options.body)});
  return {ok: true, json: async () => ({outcomes: []})};
};
await $('media-research-form').listeners.submit({preventDefault() {}});
assert.equal(lookupRequests.length, 0, 'conflicting title and genre scopes are rejected before lookup');
assert.equal($('lookup-genres').getAttribute('aria-invalid'), 'true');
assert.equal($('lookup-genres').getAttribute('aria-describedby'), 'lookup-genre-hint lookup-status');
// Repair the other field only. The genre itself never receives an edit event,
// so clearing only the edited control would leave its old error attached.
$('lookup-title').value = '';
await $('media-research-form').listeners.submit({preventDefault() {}});
assert.equal(lookupRequests.length, 1, 'repairing the conflicting title permits a local genre lookup');
assert.deepEqual(lookupRequests[0].body.jellyfin_genres, ['Animation']);
assert.equal($('lookup-status').getAttribute('data-ui-state'), 'ready');
assert.equal($('lookup-genres').getAttribute('aria-invalid'), null, 'successful cross-field repair removes the prior invalid state');
assert.equal($('lookup-genres').getAttribute('aria-describedby'), 'lookup-genre-hint', 'repair preserves permanent help and removes only the stale error link');
assert.equal(Object.hasOwn($('lookup-genres').dataset, 'errorStatus'), false);

for (const lateOutcome of ['success', 'failure']) {
  let resolveFirst, rejectFirst, resolveSecond;
  const first = new Promise((yes, no) => { resolveFirst = yes; rejectFirst = no; });
  const second = new Promise(yes => { resolveSecond = yes; });
  const signals = [];
  context.fetch = (_url, options) => {
    signals.push(options.signal);
    return signals.length === 1 ? first : second;
  };
  $('lookup-genres').value = '';
  $('lookup-title').value = 'First synthetic title';
  const firstRequest = $('media-research-form').listeners.submit({preventDefault() {}});
  assert.equal($('lookup-submit').disabled, true);
  assert.equal($('lookup-cancel').disabled, false);
  $('lookup-title').value = 'Replacement synthetic title';
  document.activeElement = $('lookup-title');
  $('media-research-form').listeners.input({target: $('lookup-title')});
  assert.equal(signals[0].aborted, true, 'editing media scope aborts the previous actual fetch signal');
  assert.equal($('lookup-submit').disabled, false, 'editing media scope releases the superseded request controls');
  assert.equal($('lookup-cancel').disabled, true);
  assert.equal($('lookup-results').getAttribute('aria-busy'), 'false');
  assert.equal(document.activeElement, $('lookup-title'), 'canceling superseded media work does not steal typing focus');
  const secondRequest = $('media-research-form').listeners.submit({preventDefault() {}});
  if (lateOutcome === 'success') resolveFirst({ok: true, json: async () => ({outcomes: [{source: 'stale synthetic provider', state: 'observed'}]})});
  else rejectFirst(new Error('late synthetic media failure'));
  await firstRequest;
  assert.equal($('lookup-status').getAttribute('data-ui-state'), 'busy', `stale media ${lateOutcome} cannot replace the new request status`);
  assert.equal($('lookup-submit').disabled, true, 'old media completion cannot release the replacement request controls');
  assert.equal($('lookup-results').getAttribute('aria-busy'), 'true');
  assert.equal($('lookup-results').textContent, '');
  resolveSecond({ok: true, json: async () => ({outcomes: []})});
  await secondRequest;
  assert.equal($('lookup-status').getAttribute('data-ui-state'), 'ready');
  assert.equal($('lookup-submit').disabled, false);
  assert.equal($('lookup-results').getAttribute('aria-busy'), 'false');
  assert.equal($('lookup-title').value, 'Replacement synthetic title');
}

console.log('PASS: actual feedback preserves local query scope; consent edits never fetch; feedback/media cancellation rejects stale results; cross-field repairs clear obsolete errors and retain help');

import {test, expect} from '@playwright/test';

// These tests inspect browser semantics and keyboard task behavior. They do not
// run NVDA, Narrator, VoiceOver, TalkBack or Orca and must not be reported as AT
// qualification. Every API response is public synthetic data; no provider I/O.
const HASH = '1111111111111111111111111111111111111111';
const catalog = {
  schema_version: 'bce-graph-preview-v1', snapshot_id: 'public-accessibility-fixture',
  digest: 'a'.repeat(64), authority_status: 'pinned-preview-not-normative',
  concepts: [
    {id: 'public:creative', label: 'Creative work', definition: 'Synthetic authored work',
      lexicalizations: [{language: 'en', term: 'creative work'}, {language: 'de', term: 'kreatives Werk'}],
      relations: [{predicate: 'related', target_id: 'public:edition'}]},
    {id: 'public:edition', label: 'Edition', definition: 'Synthetic edition',
      lexicalizations: [{language: 'en', term: 'edition'}], relations: []},
  ],
};
const records = [{
  info_hash: HASH, title: 'Synthetic film A', content_type: 'movie',
  content_source: 'jellyfin', content_id: 'public-film-a', release_year: 2024,
  languages: ['en'], seeders: 0, leechers: 0, adult_metadata: null,
  tracks: ['direct'], matched_queries: ['synthetic film'],
}];
const queryResult = {state: 'complete', read_only: true, results: records,
  track_counts: {direct: 1, concept: 0}, failures: [], warnings: []};
const fixtures = {
  '/api/v1/capabilities': {execute: true, mutation: false, cpe_write: false,
    sovereign_inference_configured: false, external_web: false},
  '/api/v1/concepts': catalog,
  '/api/v1/query/execute': queryResult,
  '/api/v1/query/plan': {state: 'planned', read_only: true, results: []},
  '/api/v1/query/observed-graph': {
    status: 'lexical_candidates_observed',
    concepts: [{id: 'public:creative', label: 'Creative work', distinct_returned_torrents: 1}],
    evidence_edges: [{concept_id: 'public:creative', info_hash: HASH, surface: 'title',
      matched_term: 'synthetic film', assessment_eligible: false}],
  },
  '/api/v1/media/lookup': {outcomes: [{source: 'jellyfin', state: 'complete',
    attribution: 'Public synthetic fixture', warnings: [], related: [],
    candidates: ['A', 'B'].map(letter => ({title: 'Synthetic film ' + letter,
      year: 2024, source: 'jellyfin', kind: 'movie', item_id: 'public-film-' + letter,
      match_basis: 'title_candidate', relationship: 'candidate'})),
  }]},
  '/api/v1/sweeps/plan': {state: 'planned', bitmagnet_queries: ['synthetic film'], searxng_queries: []},
  '/api/v1/sweeps/execute': {state: 'complete', bitmagnet_queries: ['synthetic film'], searxng_queries: []},
  '/api/v1/sweeps/limits': {remaining_searxng_json_requests: 1, max_searxng_json_requests_per_hour: 3,
    searxng_cooldown_seconds: 0, sweep_deadline_seconds: 55, client_timeout_seconds: 80,
    max_concurrent_sweeps: 1, searxng_sweep_admitted: false},
};

const json = (route, body, status = 200) => route.fulfill({status,
  contentType: 'application/json', body: JSON.stringify(body)});
const escapePattern = text => text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

async function open(page, overrides = {}) {
  const calls = [], errors = [], unknown = [];
  page.on('pageerror', error => errors.push(error.message));
  await page.route('**/api/v1/**', async route => {
    const request = route.request(), pathname = new URL(request.url()).pathname;
    calls.push({pathname, body: request.postData() ? request.postDataJSON() : null});
    if (overrides[pathname]) return overrides[pathname](route);
    if (Object.hasOwn(fixtures, pathname)) return json(route, fixtures[pathname]);
    unknown.push(pathname);
    return json(route, {error: 'Unprovided accessibility fixture route'}, 501);
  });
  await page.goto('/', {waitUntil: 'domcontentloaded'});
  await expect(page.locator('#concept-list button')).toHaveCount(2);
  await expect(page.locator('#submit')).toContainText('Run read-only search');
  return {calls, errors, unknown};
}

async function search(page) {
  await page.locator('#query').fill('synthetic film');
  await page.locator('#query').press('Enter');
  await expect(page.locator('#status')).toContainText('Unique observed: 1');
}

async function tabTo(page, selector, key = 'Tab', limit = 90) {
  for (let step = 0; step < limit; step++) {
    if (await page.locator(selector).evaluate(el => el === document.activeElement)) return;
    await page.keyboard.press(key);
  }
  throw new Error('Keyboard Tab did not reach ' + selector);
}

async function expectReflow(page, label) {
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth),
    label + ': ordinary content must not require page-wide horizontal scrolling').toBeLessThanOrEqual(1);
}

async function expectUsableFocus(locator) {
  await expect(locator).toBeFocused();
  const id = await locator.getAttribute('id');
  await expect.poll(() => locator.evaluate(el => {
    const r = el.getBoundingClientRect(), style = getComputedStyle(el);
    // Native focus scrolling may leave part of the control below the viewport.
    // Check its visible intersection and real hit target, not full containment.
    const left = Math.max(0, r.left), right = Math.min(innerWidth, r.right);
    const top = Math.max(0, r.top), bottom = Math.min(innerHeight, r.bottom);
    if (right <= left || bottom <= top) return false;
    const x = (left + right) / 2, y = (top + bottom) / 2;
    const atPoint = document.elementFromPoint(x, y);
    return (atPoint === el || el.contains(atPoint)) && style.outlineStyle !== 'none' &&
      parseFloat(style.outlineWidth) >= 2;
  }), {message: (id || 'Selected control') + ': focused control has an unobscured visible intersection and focus indication'}).toBe(true);
}

test('keyboard search exposes progress and result semantics without stealing focus [accessible-search-task]', async ({page, browserName}, info) => {
  let release;
  const pending = new Promise(resolve => { release = resolve; });
  const {calls, errors, unknown} = await open(page, {
    '/api/v1/query/execute': async route => { await pending; return json(route, queryResult); },
  });
  // Safari's default macOS preference uses Tab for fields and Option+Tab for
  // links. Exercise that native key path rather than substituting DOM focus.
  // https://support.apple.com/guide/safari/keyboard-shortcuts-and-gestures-cpsh003/mac
  const traversal = process.platform === 'darwin' && browserName === 'webkit' ? 'Alt+Tab' : 'Tab';
  info.annotations.push({type: 'keyboard-traversal', description: traversal === 'Alt+Tab'
    ? 'Native macOS WebKit: Safari Option+Tab convention; OS keyboard preference not changed.'
    : 'Tab navigation in the configured browser.'});
  await page.keyboard.press(traversal);
  await expect(page.locator('.skip-link')).toBeFocused();
  await page.keyboard.press('Enter');
  await expect(page.locator('#main-content')).toBeFocused();
  await tabTo(page, '#query', traversal);
  await page.keyboard.press('ControlOrMeta+A');
  await page.keyboard.type('synthetic film');
  await page.keyboard.press('Enter');
  await expect(page.locator('#results')).toHaveAttribute('aria-busy', 'true');
  await expect(page.locator('#query')).toBeFocused();
  const status = page.locator('#status');
  await expect(status).toHaveAttribute('role', 'status');
  await expect(status).toHaveAttribute('aria-live', 'polite');
  await expect(status).toHaveAttribute('aria-atomic', 'true');
  // Progress must not sit under aria-busy, which can defer its announcement.
  expect(await status.evaluate(el => Boolean(el.closest('[aria-busy="true"]')))).toBe(false);
  release();
  await expect(status).toContainText('Unique observed: 1');
  await expect(page.locator('#results')).toHaveAttribute('aria-busy', 'false');
  await expect(page.locator('#query')).toBeFocused();
  await expect(page.getByRole('region', {name: 'Search results', exact: true})).toBeVisible();
  await expect(page.locator('#direct')).toContainText('Synthetic film A');
  const request = calls.find(call => call.pathname === '/api/v1/query/execute');
  expect(request.body.query).toBe('synthetic film');
  expect(request.body.include_web).toBe(false);
  expect(request.body.concept_ids).toEqual([]);
  expect(errors).toEqual([]);
  expect(unknown).toEqual([]);
});

test('result and candidate action names preserve visible labels for speech input [accessible-action-names]', async ({page}) => {
  await open(page);
  await search(page);
  const inspect = page.locator('#direct .media-inspect');
  await expect(inspect).toHaveAccessibleName(/^Inspect media sources.*Synthetic film A/);
  const copy = page.locator('#direct .hash-copy');
  await page.evaluate(() => {
    Object.defineProperty(navigator, 'clipboard', {configurable: true,
      value: {writeText: async () => { throw new Error('Synthetic denied clipboard permission'); }}});
  });
  await copy.focus();
  await page.keyboard.press('Enter');
  await expect(copy.locator('..').getByRole('status')).toContainText(/copy.*(unavailable|manually)/i);
  await expect(copy).toHaveAccessibleName(new RegExp(escapePattern((await copy.innerText()).trim()), 'i'));
  await inspect.focus();
  await page.keyboard.press('Enter');
  await expect(page.locator('#lookup-title')).toBeFocused();
  await expect(page.locator('#lookup-consent')).not.toBeChecked();
  await page.locator('#lookup-submit').focus();
  await page.keyboard.press('Enter');
  const choices = page.locator('#lookup-results button');
  await expect(choices).toHaveCount(2);
  for (const [index, title] of ['Synthetic film A', 'Synthetic film B'].entries()) {
    const choice = choices.nth(index), visibleLabel = (await choice.innerText()).trim();
    await expect(choice).toHaveAccessibleName(new RegExp(escapePattern(visibleLabel)));
    await expect(choice).toHaveAccessibleName(new RegExp(title));
  }
  await choices.last().focus();
  await page.keyboard.press('Enter');
  await expect(page.locator('#lookup-external-id')).toHaveValue('public-film-B');
  await expect(page.locator('#lookup-submit')).toBeFocused();
  await expect(page.locator('#lookup-consent')).not.toBeChecked();
});

test('media validation identifies and focuses the correction without dispatch [accessible-media-recovery]', async ({page}) => {
  const {calls} = await open(page);
  await page.locator('#lookup-submit').focus();
  await page.keyboard.press('Enter');
  const title = page.locator('#lookup-title'), status = page.locator('#lookup-status');
  await expect(title).toBeFocused();
  await expect(title).toHaveAttribute('aria-invalid', 'true');
  await expect(title).toHaveAttribute('aria-describedby', /(?:^|\s)lookup-status(?:\s|$)/);
  await expect(status).toContainText(/(title|source ID|genre)/);
  expect(calls.filter(call => call.pathname === '/api/v1/media/lookup')).toEqual([]);
  await title.fill('synthetic film');
  await expect(title).not.toHaveAttribute('aria-invalid', 'true');
  await page.locator('#lookup-provider').selectOption('tmdb');
  await page.locator('#lookup-submit').focus();
  await page.keyboard.press('Enter');
  const consent = page.locator('#lookup-consent');
  await expect(consent).toBeFocused();
  await expect(consent).toHaveAttribute('aria-invalid', 'true');
  await expect(consent).toHaveAttribute('aria-describedby', /(?:^|\s)lookup-status(?:\s|$)/);
  expect(calls.filter(call => call.pathname === '/api/v1/media/lookup')).toEqual([]);
  await page.keyboard.press('Space');
  await expect(consent).toBeChecked();
  await expect(consent).not.toHaveAttribute('aria-invalid', 'true');
  await page.locator('#lookup-submit').focus();
  await page.keyboard.press('Enter');
  await expect(status).toHaveAttribute('data-ui-state', 'ready');
  expect(calls.filter(call => call.pathname === '/api/v1/media/lookup')).toHaveLength(1);
  expect(calls.find(call => call.pathname === '/api/v1/media/lookup').body.allow_external_lookup).toBe(true);
});

test('genre errors retain existing help and recover through the implicated field [accessible-genre-help]', async ({page}) => {
  const {calls} = await open(page);
  await page.locator('#lookup-provider').selectOption('tmdb');
  await page.locator('#lookup-consent').check();
  await page.locator('#lookup-genres').fill('not a numeric genre');
  await page.locator('#lookup-submit').focus();
  await page.keyboard.press('Enter');
  const genres = page.locator('#lookup-genres');
  await expect(genres).toBeFocused();
  await expect(genres).toHaveAttribute('aria-invalid', 'true');
  await expect(genres).toHaveAttribute('aria-describedby', /(?:^|\s)lookup-genre-hint(?:\s|$)/);
  await expect(genres).toHaveAttribute('aria-describedby', /(?:^|\s)lookup-status(?:\s|$)/);
  expect(calls.filter(call => call.pathname === '/api/v1/media/lookup')).toEqual([]);
  await genres.fill('28');
  await expect(genres).not.toHaveAttribute('aria-invalid', 'true');
  await expect(genres).toHaveAttribute('aria-describedby', /(?:^|\s)lookup-genre-hint(?:\s|$)/);
  await expect(page.locator('#lookup-consent')).not.toBeChecked();
  await page.locator('#lookup-consent').check();
  await page.locator('#lookup-submit').focus();
  await page.keyboard.press('Enter');
  await expect(page.locator('#lookup-status')).toHaveAttribute('data-ui-state', 'ready');
  expect(calls.find(call => call.pathname === '/api/v1/media/lookup').body.tmdb_genre_ids).toEqual([28]);
});

test('cross-field and programmatic corrections clear obsolete invalid state [accessible-cross-field-recovery]', async ({page}) => {
  await open(page);
  await page.locator('#lookup-title').fill('Conflicting title');
  await page.locator('#lookup-genres').fill('Animation');
  await page.locator('#lookup-submit').click();
  const genres = page.locator('#lookup-genres');
  await expect(genres).toHaveAttribute('aria-invalid', 'true');
  // The requested remedy can be performed in a different field. A successful
  // retry must not leave the genre control described as invalid to AT.
  await page.locator('#lookup-title').fill('');
  await page.locator('#lookup-submit').click();
  await expect(page.locator('#lookup-status')).toHaveAttribute('data-ui-state', 'ready');
  await expect(genres).not.toHaveAttribute('aria-invalid', 'true');
  await expect(genres).toHaveAttribute('aria-describedby', /(?:^|\s)lookup-genre-hint(?:\s|$)/);
  await expect(genres).not.toHaveAttribute('aria-describedby', /(?:^|\s)lookup-status(?:\s|$)/);
  await genres.fill('');
  await page.locator('#lookup-submit').click();
  await expect(page.locator('#lookup-title')).toHaveAttribute('aria-invalid', 'true');
  await search(page);
  await page.locator('#direct .media-inspect').click();
  await expect(page.locator('#lookup-title')).toHaveValue('Synthetic film A');
  await expect(page.locator('#lookup-title')).not.toHaveAttribute('aria-invalid', 'true');
  await expect(page.locator('#lookup-title')).toBeFocused();
});

test('changing a pending media lookup scope cancels obsolete candidate evidence [accessible-media-supersession]', async ({page}) => {
  const pending = [];
  const {calls} = await open(page, {'/api/v1/media/lookup': async route => {
    await new Promise(resolve => pending.push({release: resolve}));
    await json(route, {outcomes: [{source: 'jellyfin', state: 'complete',
      candidates: [{title: 'Obsolete source candidate', source: 'jellyfin',
        item_id: 'obsolete-item', kind: 'movie'}], warnings: [], related: []}]}).catch(() => {});
  }});
  await search(page);
  for (const [index, action] of ['edit', 'prefill'].entries()) {
    await page.locator('#lookup-title').fill('Previous source query');
    await page.locator('#lookup-submit').focus();
    await page.keyboard.press('Enter');
    await expect.poll(() => pending.length).toBe(index + 1);
    await expect(page.locator('#lookup-cancel')).toBeEnabled();
    if (action === 'edit') {
      await page.locator('#lookup-title').fill('Revised source query');
    } else {
      await page.locator('#direct .media-inspect').focus();
      await page.keyboard.press('Enter');
    }
    await expect(page.locator('#lookup-submit')).toBeEnabled();
    await expect(page.locator('#lookup-cancel')).toBeDisabled();
    await expect(page.locator('#lookup-results')).toHaveAttribute('aria-busy', 'false');
    await expect(page.locator('#lookup-title')).toBeFocused();
    await expect(page.locator('#lookup-title')).toHaveValue(action === 'edit' ? 'Revised source query' : 'Synthetic film A');
    pending[index].release();
    // All API traffic is intercepted; this waits for the obsolete request to
    // settle before asserting that its result did not repopulate the new scope.
    await page.waitForLoadState('networkidle');
    await expect(page.locator('#lookup-results')).not.toContainText('Obsolete source candidate');
  }
  expect(calls.filter(call => call.pathname === '/api/v1/media/lookup')).toHaveLength(2);
});

test('sweep consent is an explicit keyboard correction and never an implicit external request [accessible-sweep-consent]', async ({page}) => {
  const {calls} = await open(page);
  await page.locator('#sweep-web').check();
  await page.locator('#sweep-plan').focus();
  await page.keyboard.press('Enter');
  const consent = page.locator('#sweep-consent');
  await expect(consent).toBeFocused();
  await expect(consent).toHaveAttribute('aria-invalid', 'true');
  await expect(consent).toHaveAttribute('aria-describedby', /(?:^|\s)sweep-status(?:\s|$)/);
  await expect(page.locator('#sweep-status')).toHaveAttribute('role', 'status');
  expect(calls.filter(call => call.pathname.startsWith('/api/v1/sweeps/'))).toEqual([]);
  await page.keyboard.press('Space');
  await expect(consent).not.toHaveAttribute('aria-invalid', 'true');
  await page.locator('#sweep-plan').focus();
  await page.keyboard.press('Enter');
  await expect(page.locator('#sweep-status')).toContainText('Plan ready');
  const request = calls.find(call => call.pathname === '/api/v1/sweeps/plan');
  expect(request.body.external_consent).toBe(true);
  expect(request.body.include_searxng).toBe(true);
});

test('query and sweep cancellation restore the keyboard task without accepting late evidence [accessible-request-cancel]', async ({page}) => {
  const release = {};
  const waitForRelease = path => new Promise(resolve => { release[path] = resolve; });
  const {calls} = await open(page, {
    '/api/v1/query/execute': async route => { await waitForRelease('query'); await json(route, queryResult).catch(() => {}); },
    '/api/v1/sweeps/execute': async route => { await waitForRelease('sweep'); await json(route, fixtures['/api/v1/sweeps/execute']).catch(() => {}); },
  });
  await expect(page.locator('#query-cancel')).toBeDisabled();
  await expect(page.locator('#sweep-cancel')).toBeDisabled();
  await page.locator('#query').fill('synthetic film');
  await page.locator('#query').press('Enter');
  await expect.poll(() => Boolean(release.query)).toBe(true);
  const cancelQuery = page.locator('#query-cancel');
  await expect(cancelQuery).toBeEnabled();
  await cancelQuery.focus();
  await page.keyboard.press('Enter');
  await expect(page.locator('#status')).toContainText(/cancel/i);
  await expect(page.locator('#submit')).toBeEnabled();
  await expect(page.locator('#submit')).toBeFocused();
  await expect(page.locator('#results')).toHaveAttribute('aria-busy', 'false');
  release.query();
  await expect(page.locator('#direct')).not.toContainText('Synthetic film A');
  await page.locator('#sweep-run').focus();
  await page.keyboard.press('Enter');
  await expect.poll(() => Boolean(release.sweep)).toBe(true);
  const cancelSweep = page.locator('#sweep-cancel');
  await expect(cancelSweep).toBeEnabled();
  await cancelSweep.focus();
  await page.keyboard.press('Enter');
  await expect(page.locator('#sweep-status')).toContainText(/cancel/i);
  await expect(page.locator('#sweep-plan')).toBeEnabled();
  await expect(page.locator('#sweep-run')).toBeEnabled();
  expect(await page.evaluate(() => ['sweep-plan', 'sweep-run'].includes(document.activeElement.id))).toBe(true);
  release.sweep();
  await expect(page.locator('#sweep-status')).not.toContainText('Sweep state: complete');
  expect(calls.filter(call => call.pathname === '/api/v1/query/execute')).toHaveLength(1);
  expect(calls.filter(call => call.pathname === '/api/v1/sweeps/execute')).toHaveLength(1);
});

test('media and feedback cancellation return focus and discard late previews [accessible-panel-cancel]', async ({page}) => {
  const release = {};
  const waitForRelease = id => new Promise(resolve => { release[id] = resolve; });
  await open(page, {
    '/api/v1/media/lookup': async route => {
      await waitForRelease('lookup');
      await json(route, fixtures['/api/v1/media/lookup']).catch(() => {});
    },
    '/api/v1/discovery/feedback/preview': async route => {
      await waitForRelease('feedback');
      await json(route, {state: 'local_review_export_only', observation_count: 1,
        proposal: {candidate_label: 'late preview must not appear'}}).catch(() => {});
    },
  });
  await page.locator('#lookup-title').fill('synthetic film');
  await page.locator('#feedback-candidate').fill('synthetic candidate');
  for (const [prefix, region] of [['lookup', 'lookup-results'], ['feedback', 'feedback-output']]) {
    const submit = page.locator('#' + prefix + '-submit'), cancel = page.locator('#' + prefix + '-cancel');
    await expect(cancel).toBeDisabled();
    await submit.focus();
    await page.keyboard.press('Enter');
    await expect.poll(() => Boolean(release[prefix])).toBe(true);
    await expect(cancel).toBeEnabled();
    await expect(page.locator('#' + region)).toHaveAttribute('aria-busy', 'true');
    await cancel.focus();
    await page.keyboard.press('Enter');
    await expect(page.locator('#' + prefix + '-status')).toContainText(/cancel/i);
    await expect(submit).toBeEnabled();
    await expect(submit).toBeFocused();
    await expect(cancel).toBeDisabled();
    await expect(page.locator('#' + region)).toHaveAttribute('aria-busy', 'false');
    release[prefix]();
  }
  await expect(page.locator('#lookup-results')).not.toContainText('Synthetic film A');
  await expect(page.locator('#feedback-output')).not.toContainText('late preview must not appear');
});

test('a new query restores superseded sweep controls and preserves current evidence [accessible-sweep-supersession]', async ({page}) => {
  let release;
  const pending = new Promise(resolve => { release = resolve; });
  const {calls} = await open(page, {'/api/v1/sweeps/execute': async route => {
    await pending;
    await json(route, {state: 'complete', bitmagnet_queries: ['obsolete sweep'], searxng_queries: []}).catch(() => {});
  }});
  await page.locator('#sweep-run').focus();
  await page.keyboard.press('Enter');
  await expect.poll(() => calls.some(call => call.pathname === '/api/v1/sweeps/execute')).toBe(true);
  await expect(page.locator('#sweep-plan')).toBeDisabled();
  await search(page);
  await expect(page.locator('#sweep-plan')).toBeEnabled();
  await expect(page.locator('#sweep-run')).toBeEnabled();
  release();
  await expect(page.locator('#status')).toContainText('Unique observed: 1');
  await expect(page.locator('#sweep-details')).not.toContainText('obsolete sweep');
  await page.locator('#sweep-plan').focus();
  await page.keyboard.press('Enter');
  await expect(page.locator('#sweep-status')).toContainText('Plan ready');
});

test('native graph alternatives retain full relationships and keyboard selection focus [accessible-graph-alternative]', async ({page}) => {
  const {calls} = await open(page);
  const button = page.getByRole('group', {name: 'Accessible concept selector'}).getByRole('button', {name: 'Creative work', exact: true});
  await button.focus();
  await page.keyboard.press('Space');
  await expect(button).toBeFocused();
  await expect(button).toHaveAttribute('aria-pressed', 'true');
  await expect(page.locator('#concept-detail')).toContainText('related');
  await expect(page.locator('#concept-detail')).toContainText('Edition');
  await expect(page.locator('#concept-detail')).toContainText('kreatives Werk');
  await expect(page.locator('#use-concept')).not.toBeChecked();
  expect(calls.filter(call => call.pathname.startsWith('/api/v1/query/'))).toEqual([]);
  const node = page.locator('#concept-graph [data-id="public:edition"]');
  await node.focus();
  await page.keyboard.press('Enter');
  await expect(node).toHaveAttribute('aria-pressed', 'true');
  await expect(node).toBeFocused();
  await expect(page.getByRole('group', {name: 'Accessible concept selector'}).getByRole('button', {name: 'Edition', exact: true})).toHaveAttribute('aria-pressed', 'true');
  await search(page);
  const observed = page.getByRole('group', {name: 'Concept evidence selectors'}).getByRole('button', {name: /Creative work/});
  await observed.focus();
  await page.keyboard.press('Space');
  await expect(observed).toBeFocused();
  await expect(observed).toHaveAttribute('aria-pressed', 'true');
  await expect(page.locator('#observed-items')).toContainText('Assessment eligibility: NO');
  await expect(page.locator('[aria-modal="true"], dialog[open]')).toHaveCount(0);
});

test('320 CSS px reflow and user text spacing preserve controls and focus [accessible-reflow-spacing]', async ({page}) => {
  await page.emulateMedia({reducedMotion: 'reduce'});
  await page.setViewportSize({width: 320, height: 800});
  await open(page);
  await search(page);
  await expectReflow(page, '320 CSS pixels / 400%-equivalent width');
  // WCAG text-spacing overrides, independently of text-size enlargement.
  await page.addStyleTag({content: '* {line-height:1.5!important;letter-spacing:.12em!important;word-spacing:.16em!important} p{margin-bottom:2em!important}'});
  await expectReflow(page, '320 CSS pixels with user spacing overrides');
  await page.locator('#advanced-settings > summary').focus();
  await page.keyboard.press('Enter');
  await expect(page.locator('#advanced-settings')).toHaveAttribute('open', '');
  await expectReflow(page, 'expanded refinements with user spacing overrides');
  for (const id of ['query', 'lookup-title', 'lookup-genres', 'feedback-candidate', 'sweep-languages']) {
    const control = page.locator('#' + id);
    await control.focus();
    await expectUsableFocus(control);
    expect(await control.evaluate(el => el.getBoundingClientRect().right <= document.documentElement.clientWidth), id + ' fits the viewport').toBe(true);
  }
  await page.locator('#query').fill('reflow recovery');
  await page.locator('#query').press('Enter');
  await expect(page.locator('#status')).toContainText('Unique observed: 1');
  await expect(page.locator('#query')).toBeFocused();
});

test('200 percent text enlargement keeps task controls usable [accessible-text-enlargement]', async ({page}) => {
  await page.emulateMedia({reducedMotion: 'reduce'});
  await page.setViewportSize({width: 1280, height: 1000});
  await open(page);
  const before = await page.locator('body').evaluate(el => parseFloat(getComputedStyle(el).fontSize));
  // Simulate a larger browser root-font preference. This proves relative
  // authored sizing, not that a native browser settings panel was exercised.
  const preference = await page.addStyleTag({content: ':root{font-size:125%!important}'});
  await expect.poll(async () =>
    await page.locator('body').evaluate(el => parseFloat(getComputedStyle(el).fontSize)) / before,
  {message: 'Body follows the terminal 125% root-font preference'}).toBeCloseTo(1.25, 2);
  await preference.evaluate(el => el.remove());
  await expect.poll(() => page.locator('body').evaluate(el => parseFloat(getComputedStyle(el).fontSize)),
    {message: 'Baseline font size is restored before the independent 200% stress test'}).toBeCloseTo(before, 2);
  // A text-enlargement stress test, not a claim that browser/OS zoom was run.
  // Read all baseline sizes before changing any inheritance relationship.
  await page.evaluate(() => {
    const sizes = [...document.querySelectorAll('body, body *')]
      .filter(el => el instanceof HTMLElement)
      .map(el => [el, parseFloat(getComputedStyle(el).fontSize)]);
    for (const [el, size] of sizes) if (size > 0) el.style.setProperty('font-size', (size * 2) + 'px', 'important');
  });
  await expectReflow(page, '200 percent text enlargement');
  for (const id of ['query', 'lookup-title', 'feedback-candidate', 'sweep-languages']) {
    const control = page.locator('#' + id);
    await control.focus();
    await expectUsableFocus(control);
  }
  await search(page);
  await expect(page.locator('#direct')).toContainText('Synthetic film A');
  await expectReflow(page, 'populated results at 200 percent text enlargement');
});

test('reduced motion retains immediate feedback and keyboard focus [accessible-reduced-motion]', async ({page}) => {
  await page.emulateMedia({reducedMotion: 'reduce'});
  await open(page);
  expect(await page.evaluate(() => matchMedia('(prefers-reduced-motion: reduce)').matches)).toBe(true);
  expect(await page.locator('html').evaluate(el => getComputedStyle(el).scrollBehavior)).toBe('auto');
  const node = page.locator('#concept-graph .graph-node').last();
  const durations = await page.locator('body, #query, #concept-graph .graph-node-shape')
    .evaluateAll(elements => elements.map(el => getComputedStyle(el).transitionDuration));
  expect(durations.every(duration => duration.split(',').every(value => parseFloat(value) === 0)),
    'Reduced motion must not add even short default all-property transitions').toBe(true);
  await node.focus();
  await page.keyboard.press('Enter');
  await expect(node).toBeFocused();
  await expect(node).toHaveAttribute('aria-pressed', 'true');
  await search(page);
  await expect(page.locator('#status')).toHaveAttribute('role', 'status');
});

test('light and dark control boundaries retain nontext contrast [accessible-control-contrast]', async ({page}, info) => {
  await page.emulateMedia({forcedColors: 'none', reducedMotion: 'reduce'});
  await open(page);
  const measurements = [];
  const luminance = channels => channels.map(channel => {
    const c = channel / 255;
    return c <= .04045 ? c / 12.92 : ((c + .055) / 1.055) ** 2.4;
  }).reduce((sum, c, i) => sum + c * [.2126, .7152, .0722][i], 0);
  for (const theme of ['Light', 'Dark']) {
    await page.getByRole('radio', {name: theme + ' theme'}).check();
    await expect(page.locator('html')).toHaveAttribute('data-theme', theme.toLowerCase());
    const controls = page.locator('#query, #lookup-title, #lookup-provider, #feedback-candidate, #submit, #sweep-local-budget');
    for (let index = 0; index < await controls.count(); index++) {
      const control = controls.nth(index);
      await control.scrollIntoViewIfNeeded();
      await expect(control).toBeVisible();
      await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
      const boundary = await control.evaluate(el => {
        const style = getComputedStyle(el), rect = el.getBoundingClientRect();
        const match = style.borderLeftColor.match(/^rgba?\((.+)\)$/);
        if (!match) throw new Error('Unsupported computed boundary color: ' + style.borderLeftColor);
        const channels = match[1].split(/[,\s/]+/).filter(Boolean).map(Number);
        const ancestors = [];
        for (let adjacent = el.parentElement; adjacent; adjacent = adjacent.parentElement) {
          ancestors.push({id: adjacent.id || adjacent.tagName, background: getComputedStyle(adjacent).backgroundColor});
        }
        return {id: el.id, rgb: channels.slice(0, 3), alpha: channels[3] ?? 1,
          width: parseFloat(style.borderLeftWidth), borderStyle: style.borderLeftStyle,
          rect: rect.toJSON(), ancestors, viewport: {width: innerWidth, height: innerHeight},
          visualViewport: {scale: visualViewport.scale, left: visualViewport.offsetLeft, top: visualViewport.offsetTop}};
      });
      expect(boundary.visualViewport).toEqual({scale: 1, left: 0, top: 0});
      expect(boundary.alpha, theme + ' #' + boundary.id + ': opaque control boundary').toBe(1);
      expect(boundary.width).toBeGreaterThan(0);
      expect(boundary.borderStyle).not.toBe('none');
      const x = Math.floor(boundary.rect.left) - 2;
      const y = Math.floor((boundary.rect.top + boundary.rect.bottom) / 2);
      expect(x, 'Adjacent pixels must be inside the rendered viewport').toBeGreaterThanOrEqual(2);
      expect(y).toBeGreaterThanOrEqual(3);
      expect(y + 3).toBeLessThan(boundary.viewport.height);
      expect(await control.evaluate((el, {x, y}) => [-2, -1, 0, 1, 2].every(dy => {
        const atPoint = document.elementFromPoint(x + .5, y + dy + .5);
        return atPoint && atPoint !== el && atPoint.contains(el);
      }), {x, y}), 'Each sampled pixel must belong to an ancestor surface, not a neighbor or overlay').toBe(true);
      // Read the actual painted adjacent surface in every browser. macOS
      // mobile WebKit can report transparent ancestor CSSOM backgrounds while
      // painting opaque surfaces. Never substitute a theme token or white.
      // Keep the authored opaque boundary color: raster antialiasing does not
      // redefine that color. The PNG crop also records the rendered edge.
      // https://www.w3.org/WAI/WCAG22/Understanding/non-text-contrast.html
      const screenshot = await page.screenshot({scale: 'css', animations: 'disabled'});
      expect(await control.evaluate(el => el.getBoundingClientRect().toJSON()),
        'Control geometry must not move during pixel capture').toEqual(boundary.rect);
      const rendered = await page.evaluate(async ({encoded, x, y}) => {
        const source = new Image();
        source.src = 'data:image/png;base64,' + encoded;
        await source.decode();
        const canvas = document.createElement('canvas');
        canvas.width = source.width; canvas.height = source.height;
        const context = canvas.getContext('2d', {willReadFrequently: true});
        context.drawImage(source, 0, 0);
        const samples = [-2, -1, 0, 1, 2].map(dy =>
          Array.from(context.getImageData(x, y + dy, 1, 1).data));
        const crop = document.createElement('canvas');
        crop.width = 12; crop.height = 7;
        crop.getContext('2d').drawImage(source, x - 2, y - 3, 12, 7, 0, 0, 12, 7);
        return {samples, width: source.width, height: source.height,
          crop: crop.toDataURL('image/png').split(',')[1]};
      }, {encoded: screenshot.toString('base64'), x, y});
      expect(rendered.width).toBe(boundary.viewport.width);
      expect(rendered.height).toBe(boundary.viewport.height);
      for (const sample of rendered.samples) {
        expect(sample[3], 'Actual adjacent screenshot pixels must be opaque').toBe(255);
        expect(sample.slice(0, 3), 'The five adjacent samples must form one stable surface').toEqual(rendered.samples[0].slice(0, 3));
      }
      const a = luminance(boundary.rgb), b = luminance(rendered.samples[0].slice(0, 3));
      const ratio = (Math.max(a, b) + .05) / (Math.min(a, b) + .05);
      measurements.push({theme, ...boundary, adjacentPixels: rendered.samples, ratio});
      await info.attach(theme.toLowerCase() + '-' + boundary.id + '-painted-edge',
        {body: Buffer.from(rendered.crop, 'base64'), contentType: 'image/png'});
      expect(ratio, theme + ' #' + boundary.id + ': boundary ' + boundary.rgb +
        ' against painted adjacent ' + rendered.samples[0].slice(0, 3)).toBeGreaterThanOrEqual(3);
    }
  }
  await info.attach('control-contrast-measurements',
    {body: JSON.stringify(measurements, null, 2), contentType: 'application/json'});
});

test('forced colors retains focus and selected-state semantics [accessible-forced-colors]', async ({page}) => {
  await page.emulateMedia({forcedColors: 'active', reducedMotion: 'reduce'});
  await open(page);
  test.skip(!await page.evaluate(() => matchMedia('(forced-colors: active)').matches),
    'This browser engine does not emulate forced colors; native high-contrast qualification remains required.');
  const button = page.locator('#concept-list button').last();
  await button.focus();
  await page.keyboard.press('Enter');
  await expect(button).toHaveAttribute('aria-pressed', 'true');
  await expectUsableFocus(button);
  const colors = await button.evaluate(el => {
    const style = getComputedStyle(el);
    return {foreground: style.color, background: style.backgroundColor, adjustment: style.forcedColorAdjust};
  });
  expect(colors.foreground).not.toBe(colors.background);
  expect(colors.adjustment).not.toBe('none');
  await expect(page.locator('#concept-detail')).toContainText('Edition');
});

test('touch actions and consent labels offer usable activation areas [accessible-touch-targets]', async ({page}, info) => {
  test.skip(!info.project.use.hasTouch, 'Touch-target regression runs in the configured touch profiles.');
  await open(page);
  await search(page);
  for (const selector of ['.example-query', '.hash-copy', '#feedback-submit', '#sweep-plan', '#sweep-run']) {
    const sizes = await page.locator(selector).evaluateAll(elements => elements.map(el => {
      const rect = el.getBoundingClientRect();
      return {width: rect.width, height: rect.height};
    }));
    for (const size of sizes) {
      expect(size.width, selector + ' target width').toBeGreaterThanOrEqual(44);
      expect(size.height, selector + ' target height').toBeGreaterThanOrEqual(44);
    }
  }
  for (const id of ['lookup-consent', 'lookup-related', 'use-concept', 'sweep-consent', 'sweep-web']) {
    const label = page.locator('label').filter({has: page.locator('#' + id)});
    expect(await label.evaluate(el => el.getBoundingClientRect().height), id + ' clickable label height').toBeGreaterThanOrEqual(44);
  }
});

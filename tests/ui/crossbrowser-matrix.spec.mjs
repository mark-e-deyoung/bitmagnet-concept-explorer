import {test,expect} from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

async function open(page){
 const errors=[];
 page.on('pageerror',err=>errors.push(err.message));
 await page.goto('/',{waitUntil:'domcontentloaded'});
 await expect(page.locator('#concept-list button')).toHaveCount(2);
 return errors;
}
const reflow=async(page,label)=>{
 const excess=await page.evaluate(()=>Math.max(0,document.documentElement.scrollWidth-document.documentElement.clientWidth));
 expect(excess,label+': body must reflow without horizontal scroll').toBeLessThanOrEqual(1);
};

test('real-source-shaped synthetic journey: search, selected graph, facet source counts, screenshot [real-synthetic-journey]',async({page},info)=>{
 const errors=await open(page);
 const node=page.locator('#concept-graph .graph-node').last();
 await node.focus();
 await page.keyboard.press('Enter');
 await expect(node).toHaveAttribute('aria-pressed','true');
 await expect(node).toBeFocused();
 await expect(page.getByRole('textbox',{name:'Search phrase',exact:true})).toHaveValue('Edition');
 await page.locator('#submit').click();
 await expect(page.locator('#status')).toContainText('Unique observed: 2');
 await expect(page.locator('#status')).toHaveAttribute('data-ui-state','ready');
 await expect(page.locator('#observed-list button')).toHaveCount(2);
 await page.locator('#observed-list button').last().click();
 await expect(page.locator('#observed-items')).toContainText('Assessment eligibility: NO');
 await page.locator('#native-facet-button').click();
 await expect(page.locator('#native-facet-data details')).toHaveCount(3);
 await expect(page.locator('#native-facet-status')).toContainText('estimate');
 await page.locator('#native-facet-data details').first().locator('summary').click();
 await expect(page.locator('#native-facet-data details').first()).toHaveAttribute('open','');
 const a=await new AxeBuilder({page}).withTags(['wcag2a','wcag2aa','wcag21a','wcag21aa','wcag22a','wcag22aa']).analyze();
 expect(a.violations,a.violations.map(x=>x.id+' '+x.description).join('; ')).toEqual([]);
 expect(errors).toEqual([]);
 await reflow(page,info.project.name);
 await page.screenshot({path:'build/playwright-crossbrowser-results/'+info.project.name+'-populated.png',fullPage:true});
 await info.attach('populated-source-evidence-'+info.project.name,{path:'build/playwright-crossbrowser-results/'+info.project.name+'-populated.png',contentType:'image/png'});
});

test('theme, reduced motion and multilingual script content retain semantic parity [theme-and-motion-parity]',async({page})=>{
 await open(page);
 await page.getByRole('radio',{name:'Dark theme'}).check();
 await expect(page.locator('html')).toHaveAttribute('data-theme','dark');
 await page.reload();
 await expect(page.getByRole('radio',{name:'Dark theme'})).toBeChecked();
 await page.getByRole('radio',{name:'System theme'}).check();
 await expect(page.locator('html')).not.toHaveAttribute('data-theme',/.+/);
 await page.emulateMedia({colorScheme:'dark',reducedMotion:'reduce'});
 const dark=await page.locator('#concept-graph .graph-node-shape').first().evaluate(el=>getComputedStyle(el).fill);
 await page.emulateMedia({colorScheme:'light',reducedMotion:'reduce'});
 const light=await page.locator('#concept-graph .graph-node-shape').first().evaluate(el=>getComputedStyle(el).fill);
 expect(dark).not.toBe(light);
 const selected=page.locator('#concept-list button').first();
 await selected.click();
 await expect(selected).toHaveAttribute('aria-pressed','true');
 await expect(page.locator('#terms')).toHaveValue(/kreatives Werk/);
 await expect(page.locator('#terms')).toHaveValue(/งานสร้างสรรค์/);
 await expect(page.getByRole('textbox',{name:'Search phrase',exact:true})).toHaveValue('Creative work');
 await reflow(page,'system media+language');
});

test('narrow equivalent 400%-reflow and keyboard traversable graph',async({page})=>{
 await page.setViewportSize({width:320,height:700});
 await open(page);
 await reflow(page,'320 CSS px / 400%-equivalent reflow');
 const region=page.getByRole('region',{name:'Scrollable provisional concept diagram'});
 const dims=await region.evaluate(el=>({width:el.clientWidth,scroll:el.scrollWidth}));
 expect(dims.scroll).toBeGreaterThan(dims.width);
 await region.focus();
 await expect(region).toBeFocused();
 await region.press('ArrowRight');
 const first=page.locator('#concept-graph .graph-node').first();
 await first.focus();
 await page.keyboard.press(' ');
 await expect(first).toHaveAttribute('aria-pressed','true');
 await expect(first).toBeFocused();
});

test('source unavailable and blocked external consent cannot impersonate empty results [source-unavailable-and-consent]',async({page})=>{
 await open(page);
 await page.route('**/api/v1/query/execute',route=>route.fulfill({
  status:503,contentType:'application/json',body:JSON.stringify({error:'synthetic upstream is unavailable'})}));
 await page.getByRole('textbox',{name:'Search phrase',exact:true}).fill('creative work');
 await page.locator('#submit').click();
 await expect(page.locator('#status')).toHaveAttribute('data-ui-state','error');
 await expect(page.locator('#status')).toContainText('Query failed');
 await expect(page.locator('#details')).toContainText('No absence inference');
 await page.locator('#sweep-web').check();
 await page.locator('#sweep-plan').click();
 await expect(page.locator('#sweep-status')).toContainText('requires a separate consent');
 await expect(page.locator('#sweep-status')).not.toContainText('No results found');
});

test('navigation, skip-link, example queries, and JSON API links integrity [navigation-skip-link-and-api-links]',async({page,request})=>{
 await open(page);
 // Skip link keyboard reveal and activation
 await page.keyboard.press('Tab');
 const skip=page.locator('.skip-link');
 await expect(skip).toBeFocused();
 await page.keyboard.press('Enter');
 await expect(page.locator('#main-content')).toBeFocused();

 // Brand link accessible name matches visible text
 const brand=page.locator('.product-brand');
 await expect(brand).toHaveAttribute('href','#top');
 await expect(brand).toContainText('Concept Explorer');

 // Primary nav anchors point to present targets
 for(const hash of ['#search','#results','#concepts','#sweep']){
  const navItem=page.locator(`.primary-nav a[href="${hash}"]`);
  await expect(navItem).toHaveCount(1);
  await expect(page.locator(hash)).toBeVisible();
 }

 // Example query buttons populate search input without auto-submitting
 for(const item of [{query:'Fedora'},{query:'A Princess of Mars'},{query:'Ubuntu'}]){
  const btn=page.locator(`.examples button[data-query="${item.query}"]`);
  await btn.click();
  await expect(page.locator('#query')).toHaveValue(item.query);
  await expect(page.locator('#query')).toBeFocused();
 }

 // JSON API links have valid hrefs and return expected schema contracts
 const capRes=await request.get('/api/v1/capabilities');
 expect(capRes.status()).toBe(200);
 const capJson=await capRes.json();
 expect(capJson.execute).toBe(true);

 const concRes=await request.get('/api/v1/concepts');
 expect(concRes.status()).toBe(200);
 const concJson=await concRes.json();
 expect(concJson.concepts.length).toBeGreaterThan(0);
});

test('advanced search options serialize exact request payload and reject unpushed facet filters [advanced-search-payload-verification]',async({page})=>{
 await open(page);
 const details=page.locator('#advanced-settings');
 await details.locator('summary').click();
 await expect(details).toHaveAttribute('open','');

 await page.locator('#query').fill('Advanced payload inspection');
 await page.locator('#terms').fill('first line expansion\nsecond line expansion');
 await page.locator('#languages').fill('en, de, th');
 await page.locator('#media').selectOption('software');
 await page.locator('#minseeders').fill('5');
 await page.locator('#minleechers').fill('2');
 await page.locator('#adult').selectOption('no');
 await page.locator('#sort').selectOption('seeders_desc');
 await page.locator('#group').selectOption('media_type');
 await page.locator('#web').check();

 let capturedPayload=null;
 await page.route('**/api/v1/query/execute',async route=>{
  capturedPayload=route.request().postDataJSON();
  await route.fulfill({
   status:200,contentType:'application/json',
   body:JSON.stringify({
    state:'complete',read_only:true,results:[],track_counts:{direct:0,concept:0},failures:[],warnings:[]
   })
  });
 });

 await page.locator('#submit').click();
 await expect(page.locator('#status')).toContainText('Unique observed: 0');

 expect(capturedPayload).not.toBeNull();
 expect(capturedPayload.query).toBe('Advanced payload inspection');
 expect(capturedPayload.expansions).toEqual([
  {term:'first line expansion',provenance:'manual-unverified'},
  {term:'second line expansion',provenance:'manual-unverified'}
 ]);
 expect(capturedPayload.filters.languages).toEqual(['en','de','th']);
 expect(capturedPayload.filters.content_types).toEqual(['software']);
 expect(capturedPayload.filters.min_seeders).toBe(5);
 expect(capturedPayload.filters.min_leechers).toBe(2);
 expect(capturedPayload.filters.adult_metadata).toBe('no');
 expect(capturedPayload.sort).toBe('seeders_desc');
 expect(capturedPayload.group_by).toBe('media_type');
 expect(capturedPayload.include_web).toBe(true);

 // Pushdown guard: active min-peer and non-any adult filter must block native facet aggregation
 await page.locator('#native-facet-button').click();
 await expect(page.locator('#native-facet-status')).toContainText('Clear min-peer and source-adult filters first');
 await expect(page.locator('#native-facet-status')).toHaveAttribute('data-ui-state','blocked');

 // Once filters are cleared, facet aggregation proceeds
 await page.locator('#minseeders').fill('');
 await page.locator('#minleechers').fill('');
 await page.locator('#adult').selectOption('any');
 await page.locator('#native-facet-button').click();
 await expect(page.locator('#native-facet-status')).toContainText('estimate');
 await expect(page.locator('#native-facet-data details.facet-group')).toHaveCount(3);
});

test('concept filter input, inclusion checkbox opt-in, and keyboard node selection [concept-filter-and-selection-optin]',async({page})=>{
 await open(page);
 // Concept filter matches 'Edition'
 await page.locator('#concept-filter').fill('edition');
 await expect(page.locator('#concept-detail')).toContainText('Selected provisional concept: Edition');
 await expect(page.locator('#query')).toHaveValue('Edition');

 // Include concept lexicalizations opt-in
 await page.locator('#use-concept').check();
 let captured=null;
 await page.route('**/api/v1/query/execute',async route=>{
  captured=route.request().postDataJSON();
  await route.fulfill({
   status:200,contentType:'application/json',
   body:JSON.stringify({state:'complete',read_only:true,results:[],track_counts:{direct:0,concept:0},failures:[],warnings:[]})
  });
 });
 await page.locator('#submit').click();
 expect(captured.concept_ids).toEqual(['public:edition']);

 // Opt out
 await page.locator('#use-concept').uncheck();
 await page.locator('#submit').click();
 expect(captured.concept_ids).toEqual([]);

 // Accessible concept list alternative button selection
 const creativeBtn=page.locator('#concept-list button').filter({hasText:'Creative work'});
 await creativeBtn.click();
 await expect(creativeBtn).toHaveAttribute('aria-pressed','true');
 await expect(page.locator('#query')).toHaveValue('Creative work');
});

test('infohash clipboard copy success, fallback permission handling, and sovereign model hypotheses [hash-clipboard-copy-and-sovereign-hypotheses]',async({page})=>{
 await open(page);
 await page.locator('#submit').click();
 await expect(page.locator('#status')).toContainText('Unique observed: 2');

 const copyBtn=page.locator('.hash-copy').first();
 await expect(copyBtn).toBeVisible();

 // Trigger clipboard copy interaction: handles both headless container fallback
 // ('Select text') and writeText resolution ('Copied') across varying browser OS grants.
 await copyBtn.click();
 const copyStatus = await copyBtn.textContent();
 expect(['Copied', 'Select text']).toContain(copyStatus);

 const copyNotice = page.locator('.hash-row .sr-only[role="status"]').first();
 await expect(copyNotice).toContainText(/Infohash copied|Copy unavailable/);

 // Sovereign hypothesis opt-in button
 const sovereignBtn=page.locator('#observed-items button').first();
 await expect(sovereignBtn).toContainText('Ask sovereign model for hypotheses');
 await sovereignBtn.click();
 await expect(page.locator('#observed-items pre.muted').first()).toContainText('UNVERIFIED MODEL HYPOTHESES');
});

test('multilingual sweep planning, bounded execution, budget limits, and consent authorization enforcement [multilingual-sweep-plan-and-execution]',async({page})=>{
 await open(page);
 await page.locator('#sweep-languages').fill('en,de,th');
 await page.locator('#sweep-local-budget').fill('6');
 await page.locator('#sweep-external-budget').fill('2');

 // Limits lookup
 await page.locator('#sweep-limits').click();
 await expect(page.locator('#sweep-status')).toContainText('SearXNG JSON budget');
 await expect(page.locator('#sweep-details')).toContainText('max_searxng_json_requests_per_hour');

 // Plan preview without external search
 await page.locator('#sweep-plan').click();
 await expect(page.locator('#sweep-status')).toContainText('Plan ready');
 await expect(page.locator('#sweep-details')).toContainText('bitmagnet_queries');

 // Execution with external search and consent
 await page.locator('#sweep-web').check();
 await page.locator('#sweep-consent').check();
 await page.locator('#sweep-run').click();
 await expect(page.locator('#sweep-status')).toContainText('Sweep state: complete');
});

test('secondary diagnostics disclosure, raw evidence inspection, and WCAG axe compliance on populated state [diagnostics-disclosure-and-wcag-axe]',async({page})=>{
 await open(page);
 await page.locator('#submit').click();
 await expect(page.locator('#status')).toHaveAttribute('data-ui-state','ready');

 // Diagnostics disclosure
 const diagDetails=page.locator('#technical-evidence details.advanced');
 await diagDetails.locator('summary').click();
 await expect(diagDetails).toHaveAttribute('open','');
 await expect(page.locator('#details')).toContainText('track_counts');

 // Observed graph node selection
 const observedNode=page.locator('#observed-graph .graph-node').first();
 await observedNode.click();
 await expect(observedNode).toHaveAttribute('aria-pressed','true');

 // Axe WCAG full scan on populated state
 const a11y=await new AxeBuilder({page}).withTags(['wcag2a','wcag2aa','wcag21a','wcag21aa','wcag22a','wcag22aa']).analyze();
 expect(a11y.violations,a11y.violations.map(x=>x.id+' '+x.description).join('; ')).toEqual([]);
 await reflow(page,'diagnostics populated');
});

import {test,expect} from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

const catalog={
 schema_version:'bce-graph-preview-v1',snapshot_id:'synthetic-r2-interaction',
 digest:'a'.repeat(64),authority_status:'pinned-preview-not-normative',
 concepts:[
  {id:'public:creative',label:'Creative work',definition:'Synthetic authored-work type',
   lexicalizations:[{language:'en',term:'creative work'},{language:'de',term:'kreatives Werk'}],
   relations:[{predicate:'related',target_id:'public:edition'}]},
  {id:'public:edition',label:'Edition',definition:'Synthetic manifestation type',
   lexicalizations:[{language:'en',term:'edition'}],
   relations:[]}
 ]
};
const hash='1111111111111111111111111111111111111111';
const records=[
 {info_hash:hash,title:'Creative work — public synthetic fixture',content_type:'ebook',languages:['en'],
  seeders:0,leechers:0,adult_metadata:null,tracks:['direct','concept'],matched_queries:['creative work','edition']}
];
const observed={
 status:'lexical_candidates_observed',
 concepts:[{id:'public:creative',label:'Creative work',distinct_returned_torrents:1},
  {id:'public:edition',label:'Edition',distinct_returned_torrents:1}],
 evidence_edges:[
  {concept_id:'public:creative',info_hash:hash,surface:'title',matched_term:'creative work',
   assessment_eligible:false,evidence_role:'candidate_lexical_match'},
  {concept_id:'public:edition',info_hash:hash,surface:'title',matched_term:'edition',
   assessment_eligible:false,evidence_role:'candidate_lexical_match'}
 ]
};
const facets={
 total_count:172,total_count_is_estimate:true,
 scope:'native_corpus_metadata_aggregate_not_cpe_concepts',
 groups:{
  contentType:[{label:'Ebook',count:54,is_estimate:false},{label:'Software',count:88,is_estimate:true}],
  language:[{label:'English',count:70,is_estimate:true},{label:'German',count:22,is_estimate:false}],
  genre:[{label:'Not classified',count:14,is_estimate:true}]
 }
};
let endpointCalls;
async function setup(page){
 endpointCalls=[];
 await page.route('**/api/v1/**',async route=>{
  const path=new URL(route.request().url()).pathname;
  endpointCalls.push(path);
  const response=path==='/api/v1/capabilities'
   ? {execute:true,mutation:false,cpe_write:false}
   :path==='/api/v1/concepts'?catalog
   :path==='/api/v1/query/execute'?{state:'complete',results:records,track_counts:{direct:1,concept:1},failures:[],warnings:[]}
   :path==='/api/v1/query/observed-graph'?observed
   :path==='/api/v1/query/facets'?facets
   :path==='/api/v1/sweeps/limits'
    ? {remaining_searxng_json_requests:1,max_searxng_json_requests_per_hour:3,searxng_cooldown_seconds:0,
      sweep_deadline_seconds:55,client_timeout_seconds:80,max_concurrent_sweeps:1,searxng_sweep_admitted:false}
   :{state:'complete',bitmagnet_queries:[],searxng_queries:[]};
  await route.fulfill({status:200,contentType:'application/json',body:JSON.stringify(response)});
 });
 await page.goto('/',{waitUntil:'domcontentloaded'});
 await expect(page.locator('#concept-list button')).toHaveCount(2);
}
async function search(page){
 await page.getByRole('textbox',{name:'Search phrase',exact:true}).fill('creative work');
 await page.getByRole('button',{name:'Run read-only search'}).click();
 await expect(page.locator('#status')).toContainText('Unique observed: 1');
 await expect(page.locator('#observed-list button')).toHaveCount(2);
}

test('both graph selections stay native, keyboard-accessible and visually theme-derived',async({page},testInfo)=>{
 await setup(page);
 const graph=page.locator('#concept-graph');
 const nodes=graph.locator('.graph-node');
 await expect(nodes).toHaveCount(2);
 await expect(nodes.filter({has:page.locator('.graph-node-shape')})).toHaveCount(2);
 await expect(graph).toHaveAttribute('role','group');
 await expect(graph.locator('[fill]')).toHaveCount(0);
 await expect(graph.locator('[stroke]')).toHaveCount(0);
 await expect(nodes.filter({'hasText':'Creative work'})).toHaveAttribute('aria-pressed','true');
 await page.getByRole('radio',{name:'Light theme'}).check();
 const firstColors=await page.locator('#concept-graph .graph-node-shape').first().evaluate(el=>({
  fill:getComputedStyle(el).fill,stroke:getComputedStyle(el).stroke
 }));
 await page.getByRole('radio',{name:'Dark theme'}).check();
 const darkColors=await page.locator('#concept-graph .graph-node-shape').first().evaluate(el=>({
  fill:getComputedStyle(el).fill,stroke:getComputedStyle(el).stroke
 }));
 expect(darkColors.fill).not.toBe(firstColors.fill);
 expect(darkColors.stroke).not.toBe(firstColors.stroke);
 const secondary=nodes.filter({hasText:'Edition'});
 await secondary.focus();
 await page.keyboard.press('Enter');
 await expect(secondary).toHaveAttribute('aria-pressed','true');
 await expect(page.locator('#concept-list button').filter({hasText:'Edition'})).toHaveAttribute('aria-pressed','true');
 await expect(page.locator('#concept-detail')).toContainText('no query was executed');
 expect(endpointCalls.filter(p=>p==='/api/v1/query/execute')).toHaveLength(0);
 await page.screenshot({path:'build/playwright-results/r2-interactive-dark.png',fullPage:true});
 await testInfo.attach('dark-populated-provisional-graph',{path:'build/playwright-results/r2-interactive-dark.png',contentType:'image/png'});
});

test('observed graph keeps evidence separate from accepted CPE authority',async({page})=>{
 await setup(page);await search(page);
 const graph=page.locator('#observed-graph');
 await expect(graph.locator('.graph-node')).toHaveCount(2);
 await expect(graph.locator('.graph-node[aria-pressed=true]')).toHaveCount(1);
 await expect(page.locator('#observed-list button[aria-pressed=true]')).toHaveCount(1);
 await page.locator('#observed-list button').filter({hasText:'Edition'}).click();
 await expect(graph.locator('.graph-node[data-id="public:edition"]')).toHaveAttribute('aria-pressed','true');
 await expect(page.locator('#observed-items')).toContainText('Assessment eligibility: NO');
 await expect(page.locator('#observed-status')).toContainText('Not CPE assertions');
 await expect(page.locator('#status')).toHaveAttribute('data-ui-state','ready');
});

test('facet estimates are explicit, long buckets disclose without invented filter actions',async({page})=>{
 await setup(page);
 await page.getByRole('button',{name:'Load native corpus-side counts'}).click();
 await expect(page.locator('#native-facet-status')).toContainText('172 (estimate)');
 await expect(page.locator('#native-facet-status')).toHaveAttribute('data-ui-state','ready');
 const groups=page.locator('#native-facet-data details.facet-group');
 await expect(groups).toHaveCount(3);
 const group=groups.filter({hasText:'Content type'});
 await expect(group).not.toHaveAttribute('open','');
 await group.locator('summary').focus();
 await page.keyboard.press('Enter');
 await expect(group).toHaveAttribute('open','');
 await expect(group.locator('.facet-buckets li')).toHaveCount(2);
 await expect(group.locator('.facet-amount').filter({hasText:'Estimate'})).toHaveCount(1);
 await expect(group.locator('.facet-label')).toContainText(['Ebook','Software']);
 await expect(group.locator('button')).toHaveCount(0);
});

test('full interactive graph, disclosures and source metadata pass WCAG axe at narrow viewport',async({page})=>{
 await page.setViewportSize({width:320,height:640});
 await setup(page);await search(page);
 await page.getByRole('button',{name:'Load native corpus-side counts'}).click();
 await expect(page.locator('#native-facet-data details')).toHaveCount(3);
 await expect(page.locator('#observed-graph .graph-node')).toHaveCount(2);
 const overflow=await page.evaluate(()=>document.documentElement.scrollWidth-document.documentElement.clientWidth);
 expect(overflow,'320px no document overflow').toBeLessThanOrEqual(1);
 const scroller=page.getByRole('region',{name:'Scrollable observed evidence diagram'});
 const sizes=await scroller.evaluate(e=>({client:e.clientWidth,scroll:e.scrollWidth}));
 expect(sizes.scroll).toBeGreaterThan(sizes.client);
 const result=await new AxeBuilder({page}).withTags(['wcag2a','wcag2aa','wcag21a','wcag21aa','wcag22a','wcag22aa']).analyze();
 expect(result.violations,result.violations.map(x=>x.id).join(',')).toEqual([]);
});

test('forced-colors mode retains selected graph node semantics and readable labels',async({page},testInfo)=>{
 await page.emulateMedia({forcedColors:'active',reducedMotion:'reduce'});
 await setup(page);await search(page);
 const selected=page.locator('#observed-graph .graph-node[aria-pressed=true]');
 await expect(selected).toHaveCount(1);
 await expect(page.locator('#observed-list button[aria-pressed=true]')).toHaveCount(1);
 const style=await selected.locator('.graph-node-shape').evaluate(el=>({
  stroke:getComputedStyle(el).stroke,width:getComputedStyle(el).strokeWidth
 }));
 expect(parseFloat(style.width)).toBeGreaterThanOrEqual(4);
 await page.screenshot({path:'build/playwright-results/r2-interactive-forced-colors.png',fullPage:true});
 await testInfo.attach('forced-colors-populated-graphs',{path:'build/playwright-results/r2-interactive-forced-colors.png',contentType:'image/png'});
});

test('network refusal uses error state, clears obsolete evidence and never means absence',async({page})=>{
 await setup(page);await search(page);
 await page.route('**/api/v1/query/execute',route=>route.fulfill({status:503,contentType:'application/json',body:JSON.stringify({error:'temporary source refusal'})}));
 await page.getByRole('button',{name:'Run read-only search'}).click();
 await expect(page.locator('#status')).toHaveAttribute('data-ui-state','error');
 await expect(page.locator('#status')).toContainText('Query failed');
 await expect(page.locator('#details')).toContainText('No absence inference');
 await expect(page.locator('#direct article')).toHaveCount(0);
});

test('third-party sweep consent is still explicit and failed source limits are not shown as absence',async({page})=>{
 await setup(page);
 await page.getByRole('button',{name:'View source budget'}).click();
 await expect(page.locator('#sweep-status')).toContainText('SearXNG JSON budget');
 await expect(page.locator('#sweep-status')).toHaveAttribute('data-ui-state','ready');
 await page.locator('#sweep-web').check();
 await page.getByRole('button',{name:'Preview sweep plan'}).click();
 await expect(page.locator('#sweep-status')).toContainText('requires a separate consent');
 expect(endpointCalls.filter(x=>x==='/api/v1/sweeps/plan')).toHaveLength(0);
});

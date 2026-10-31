import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

const url='/';
const catalog={schema_version:'bce-graph-preview-v1',snapshot_id:'synthetic-ui-proof',digest:'a'.repeat(64),
  authority_status:'pinned-preview-not-normative',concepts:[]};
const rows=[
 {info_hash:'1111111111111111111111111111111111111111',title:'Fedora Workstation',content_type:'software',
  languages:['en'],seeders:0,leechers:0,adult_metadata:false,tracks:['direct'],matched_queries:['Fedora']},
 {info_hash:'2222222222222222222222222222222222222222',title:'Related software candidate',content_type:'software',
  languages:['en'],seeders:1,leechers:0,adult_metadata:null,tracks:['concept'],matched_queries:['software']},
 {info_hash:'3333333333333333333333333333333333333333',title:'Fedora evidence overlap',content_type:'software',
  languages:['en'],seeders:2,leechers:0,adult_metadata:false,tracks:['direct','concept'],matched_queries:['Fedora','software']},
];
const observed={status:'lexical_candidates_observed',concepts:[],evidence_edges:[]};
async function withFixture(page) {
  await page.route('**/api/v1/**',async route=>{
    const req=route.request();const path=new URL(req.url()).pathname;
    const result=path==='/api/v1/capabilities'
      ? {execute:true,mutation:false,cpe_write:false,audiences:['human','automation','agent']}
      : path==='/api/v1/concepts'? catalog
      : path==='/api/v1/query/execute'
        ? {state:'complete',read_only:true,results:rows,track_counts:{direct:2,concept:2},failures:[],warnings:[],groups:[]}
        : path==='/api/v1/query/observed-graph'? observed
        : {state:'complete',read_only:true,results:[],failures:[]};
    await route.fulfill({status:200,contentType:'application/json',body:JSON.stringify(result)});
  });
  await page.goto(url,{waitUntil:'domcontentloaded'});
  await expect(page.getByRole('status').filter({hasText:'Read-only Bitmagnet execution configured'})).toBeVisible();
}
const noHorizontalOverflow=async(page,label)=>{
  const width=await page.evaluate(()=>document.documentElement.scrollWidth-document.documentElement.clientWidth);
  expect(width,label+' horizontal overflow').toBeLessThanOrEqual(1);
};

for(const size of [{name:'desktop',width:1440,height:900},{name:'narrow',width:390,height:844}]){
  test(size.name+' primary search and results stay task-first',async({page},testInfo)=>{
    await page.setViewportSize({width:size.width,height:size.height});
    await withFixture(page);
    await expect(page.getByRole('heading',{name:"Find what's in the index."})).toBeVisible();
    await expect(page.getByRole('navigation',{name:'Primary'})).toBeVisible();
    await expect(page.getByRole('textbox',{name:'Search phrase',exact:true})).toBeVisible();
    await expect(page.locator('details#advanced-settings')).not.toHaveAttribute('open','');
    await page.getByRole('button',{name:'Fedora',exact:true}).click();
    await expect(page.getByRole('textbox',{name:'Search phrase',exact:true})).toHaveValue('Fedora');
    await page.getByRole('button',{name:'Run read-only search'}).click();
    await expect(page.locator('#status')).toContainText('Unique observed: 3');
    await expect(page.locator('#direct article')).toHaveCount(1);
    await expect(page.locator('#concept article')).toHaveCount(1);
    await expect(page.locator('#overlap article')).toHaveCount(1);
    await expect(page.locator('#direct article dl dt').filter({hasText:'Seeders'})).toHaveCount(1);
    await expect(page.locator('#direct article .hash-row code')).toHaveText('1111111111111111111111111111111111111111');
    await expect(page.locator('#native-facet-status')).toHaveText('Counts not requested for this search.');
    await expect(page.locator('#sweep-status')).toContainText('No sweep run for this search');
    await expect(page.locator('#concept-graph')).toBeHidden();
    await expect(page.locator('#observed-graph')).toBeHidden();
    await noHorizontalOverflow(page,size.name);
    await page.screenshot({path:'build/playwright-results/'+size.name+'-light.png',fullPage:true});
    await testInfo.attach(size.name+' full-page UI',{path:'build/playwright-results/'+size.name+'-light.png',contentType:'image/png'});
  });
}

test('three native themes persist, follow system mode, and remain keyboard reachable',async({page})=>{
 await withFixture(page);
 const group=page.getByRole('group',{name:'Theme'});
 await expect(group).toBeVisible();
 await page.getByRole('radio',{name:'Dark theme'}).check();
 await expect(page.locator('html')).toHaveAttribute('data-theme','dark');
 await page.reload();
 await expect(page.getByRole('radio',{name:'Dark theme'})).toBeChecked();
 await page.getByRole('radio',{name:'Light theme'}).check();
 await expect(page.locator('html')).toHaveAttribute('data-theme','light');
 await page.getByRole('radio',{name:'System theme'}).check();
 await expect(page.locator('html')).not.toHaveAttribute('data-theme',/.+/);
 for(const label of ['System theme','Light theme','Dark theme']){
  const target=page.getByRole('radio',{name:label});
  await expect(target).toHaveCount(1);
  const parent=target.locator('..');
  const box=await parent.boundingBox();
  expect(box.width,label+' target width').toBeGreaterThanOrEqual(44);
  expect(box.height,label+' target height').toBeGreaterThanOrEqual(44);
 }
});

test('advanced filters are discoverable and native controls retain labeling',async({page})=>{
 await withFixture(page);
 const details=page.locator('#advanced-settings');
 await expect(details).not.toHaveAttribute('open','');
 await page.getByText('Refine search · languages, concepts and filters').click();
 await expect(details).toHaveAttribute('open','');
 await page.getByLabel('Languages (comma separated IDs)').fill('en,de,th');
 await expect(page.getByLabel('Bitmagnet media type')).toBeVisible();
 await expect(page.getByLabel('Maximum SearXNG queries (1–3)')).toHaveAttribute('max','3');
});

test('automated browser axe WCAG 2.2 A/AA checks and visible focus on primary journey',async({page})=>{
 await withFixture(page);
 const a11y=await new AxeBuilder({page}).withTags(['wcag2a','wcag2aa','wcag21a','wcag21aa','wcag22a','wcag22aa']).analyze();
 expect(a11y.violations,a11y.violations.map(v=>v.id+': '+v.description).join('; ')).toEqual([]);
 await page.keyboard.press('Tab');
 await expect(page.locator('.skip-link')).toBeFocused();
 const outline=await page.locator('.skip-link').evaluate(el=>getComputedStyle(el).outlineWidth);
 expect(parseInt(outline,10)).toBeGreaterThanOrEqual(2);
});

test('mobile page can reflow at 320px with no horizontal overflow',async({page})=>{
 await page.setViewportSize({width:320,height:640});
 await withFixture(page);
 await noHorizontalOverflow(page,'320 CSS px');
 const footer=page.locator('.site-footer');
 await footer.scrollIntoViewIfNeeded();
 await noHorizontalOverflow(page,'320 CSS px footer');
});

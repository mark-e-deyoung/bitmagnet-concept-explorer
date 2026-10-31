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

test('real-source-shaped synthetic journey: search, selected graph, facet source counts, screenshot',async({page},info)=>{
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

test('theme, reduced motion and multilingual script content retain semantic parity',async({page})=>{
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
 selected.click();
 await expect(page.locator('#terms')).toContainText('');
 await expect(page.locator('#terms')).toHaveValue(/kreatives Werk/);
 await expect(page.locator('#terms')).toHaveValue(/งานสร้างสรรค์/);
 await expect(page.getByRole('textbox',{name:'Search phrase',exact:true})).toHaveValue('Creative work');
 await reflow(page,'system media+language');
});

test('narrow equivalent 400%-reflow and keyboard traversable graph',async({page})=>{
 await page.setViewportSize({width:320,height:700}); // 1280px CSS desktop at 400% zoom equivalent.
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

test('source unavailable and blocked external consent cannot impersonate empty results',async({page})=>{
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

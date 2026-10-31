#!/usr/bin/env node
// Actual Lighthouse navigation + interaction timespan + populated-state snapshot.
// All requests resolve to an isolated public/synthetic localhost fixture.
import puppeteer from 'puppeteer-core';
import {startFlow,desktopConfig} from 'lighthouse';
import {writeFile,mkdir} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
import path from 'node:path';
import {execFileSync} from 'node:child_process';

const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'../..');
const out=path.join(root,'build/lighthouse');
const url='http://127.0.0.1:'+(process.env.UX_FIXTURE_PORT||'4175')+'/';
const chrome=process.env.CHROME_PATH;
if(!chrome)throw Error('CHROME_PATH required: use exact installed Playwright Chromium binary');
await mkdir(out,{recursive:true});
const min={desktop:{performance:80,accessibility:90,'best-practices':85},
 mobile:{performance:70,accessibility:90,'best-practices':85}};
const findings=[];
async function test(mode){
 const browser=await puppeteer.launch({executablePath:chrome,headless:true,
  args:['--no-sandbox','--disable-dev-shm-usage','--disable-gpu']});
 let flow;
 try{
  const page=await browser.newPage();
  const isDesktop=mode==='desktop';
  await page.setViewport(isDesktop?{width:1440,height:900,deviceScaleFactor:1}
   :{width:390,height:844,deviceScaleFactor:2,isMobile:true,hasTouch:true});
  flow=await startFlow(page,{name:'Bitmagnet Concept Explorer '+mode+' localhost synthetic UX',
   ...(isDesktop?{config:desktopConfig}:{})});
  await flow.navigate(url,{stepName:'Initial read-only research UI'});
  await page.waitForSelector('#concept-list button',{timeout:12000});
  await flow.startTimespan({stepName:'Explore concept then query and inspect source facets'});
  await page.click('#concept-list button:last-child');
  await page.focus('#query');
  await page.click('#submit');
  await page.waitForFunction(()=>document.querySelector('#status')?.textContent?.includes('Unique observed: 2'),
   {timeout:12000});
  await page.click('#native-facet-button');
  await page.waitForSelector('#native-facet-data details.facet-group',{timeout:12000});
  await page.click('#native-facet-data details.facet-group summary');
  await flow.endTimespan();
  await flow.snapshot({stepName:'Populated evidence graph, source estimates, expanded facets'});
  const report=await flow.createFlowResult();
  const html=await flow.generateReport();
  await writeFile(path.join(out,mode+'-flow.json'),JSON.stringify(report,null,2));
  await writeFile(path.join(out,mode+'-flow.html'),html);
  await page.screenshot({path:path.join(out,mode+'-populated.png'),fullPage:true});
  const steps=report.steps||[];
  const nav=steps.find(x=>x.lhr?.categories?.performance?.score!=null);
  const snap=steps.filter(x=>x.lhr?.categories?.accessibility?.score!=null).at(-1);
  if(!nav)throw Error(mode+': Lighthouse navigation report missing, cannot claim score');
  if(!snap)throw Error(mode+': Lighthouse populated snapshot missing, cannot claim accessibility');
  const scores={};
  for(const category of ['performance','accessibility','best-practices','seo']){
   const navScore=nav.lhr.categories?.[category]?.score;
   scores[category]=navScore==null?null:Math.round(navScore*100);
  }
  const snapScores={};
  for(const category of ['accessibility','best-practices']){
   const value=snap.lhr.categories?.[category]?.score;
   snapScores[category]=value==null?null:Math.round(value*100);
  }
  const audits=nav.lhr.audits||{};
  const metrics={};
  for(const [name,id] of [['lcp','largest-contentful-paint'],['cls','cumulative-layout-shift'],
   ['tbt','total-blocking-time'],['fcp','first-contentful-paint'],['speed-index','speed-index']]){
   metrics[name]=audits[id]?.numericValue??null;
  }
  const invalid=[];
  for(const [category,threshold] of Object.entries(min[mode])){
   if(scores[category]===null||scores[category]<threshold){
    invalid.push('Navigation '+category+': '+scores[category]+' below '+threshold);
   }
  }
  if(snapScores.accessibility==null||snapScores.accessibility<90)
   invalid.push('Populated snapshot accessibility: '+snapScores.accessibility+' below 90');
  if(snapScores['best-practices']!=null&&snapScores['best-practices']<85)
   invalid.push('Populated snapshot best-practices: '+snapScores['best-practices']+' below 85');
  // Aggregate Lighthouse score 100 can hide audit-specific failures. Never
  // accept failures in Label-in-Name and core WCAG interaction semantics.
  const critical=['label-content-name-mismatch','aria-allowed-attr','aria-required-attr',
   'aria-required-children','color-contrast','button-name','link-name',
   'aria-hidden-focus','focus-traps','duplicate-id-aria'];
  for(const [name,lhr] of [['navigation',nav.lhr],['populated snapshot',snap.lhr]]){
   for(const id of critical){
    const audit=lhr.audits?.[id];
    if(audit?.score===0){
     invalid.push(name+' critical accessibility audit '+id+' FAIL: '+(audit.title||''));
    }
   }
  }
  // Lighthouse is laboratory/fixture evidence, not a CrUX field result.
  // SEO intentionally informational only: this is a noindex RDTE preview.
  const problems=Object.values(snap.lhr.audits||{}).filter(x=>
   x.score!==null&&x.score<1&&['binary','numeric'].includes(x.scoreDisplayMode||'')&&
   x.details?.type==='table').slice(0,20).map(x=>({id:x.id,title:x.title,score:x.score}));
  findings.push({mode,scores,snapScores,metrics,invalid,problems,
   steps:steps.map(s=>({name:s.name,gatherMode:s.lhr?.gatherMode||s.gatherMode||null}))});
  console.log('LIGHTHOUSE_UX '+JSON.stringify({mode,scores,snapScores,metrics,invalid}));
 }finally{await browser.close();}
}
for(const mode of ['desktop','mobile']){
 try{await test(mode);}catch(err){findings.push({mode,invalid:[String(err?.stack||err)]});console.error('LIGHTHOUSE_FIRST_FAILURE '+mode,err?.stack||err);}
}
const checkedOutSha=execFileSync('git',['rev-parse','HEAD'],{cwd:root,encoding:'utf8'}).trim();
await writeFile(path.join(out,'summary.json'),JSON.stringify({sha:checkedOutSha,github_event_sha:process.env.GITHUB_SHA||null,
  fixture:'public-ux-fixture-only',noindex:true,results:findings,thresholds:min,
  note:'Synthetic local laboratory Chrome. SEO noindex intentionally not gated; no real user, assistive hardware or production SLA.'},null,2));
const bad=findings.flatMap(x=>x.invalid||[]);
if(bad.length){console.error('LIGHTHOUSE_FAIL '+JSON.stringify(bad));process.exitCode=1;}
else console.log('LIGHTHOUSE_PASS desktop and mobile navigation, timespan, and populated snapshot');

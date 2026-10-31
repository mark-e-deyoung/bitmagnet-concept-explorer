// Actual Chromium input-event/pointer-actuation qualification using CDP.
// Chromium with classic vertical scrollbars may report clientWidth 17 px less than the\n// layout viewport; horizontal overflow compares scrollWidth with innerWidth.\n// Native Node and Chrome only; no Playwright or npm dependency, no network
// navigation, no source write, no CPE/rig operation, no acceptance promotion.
import {spawn} from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import {setTimeout as sleep} from 'node:timers/promises';

const [htmlPath,jsonPath,chromeExec=process.env.CHROME||'chromium']=process.argv.slice(2);
if(!htmlPath||!jsonPath)throw Error('usage: browser_cdpsmoke.mjs HTML JSON [CHROME]');
if(typeof WebSocket!=='function')throw Error('Node >=22 WebSocket required');
const expect=JSON.parse(fs.readFileSync(jsonPath,'utf8'));
if(expect.query!=='right shoulder'||expect.visible_count!==2||expect.candidate_count!==64)throw Error('unexpected original source fixture');
const tmp=fs.mkdtempSync(path.join(os.tmpdir(),'source-cdp-'));
const chrome=spawn(chromeExec,['--headless=new','--no-sandbox','--disable-gpu','--disable-dev-shm-usage',
 '--no-first-run','--user-data-dir='+tmp,'--remote-debugging-port=0','about:blank'],{stdio:'ignore'});
let ws;let id=0;const pending=new Map();
function send(method,params={}){
 const n=++id;
 const ready=new Promise((resolve,reject)=>pending.set(n,{resolve,reject}));
 ws.send(JSON.stringify({id:n,method,params}));
 return ready;
}
try{
 let port;
 for(let i=0;i<100;i++){
  if(chrome.exitCode!==null)throw Error('browser exited '+chrome.exitCode);
  try{port=Number(fs.readFileSync(path.join(tmp,'DevToolsActivePort'),'utf8').split('\n')[0]);if(port)break}catch{}
  await sleep(100);
 }
 if(!port)throw Error('CDP startup timeout');
 const tabs=await(await fetch('http://127.0.0.1:'+port+'/json')).json();
 const tab=tabs.find(t=>t.type==='page');if(!tab)throw Error('CDP page missing');
 ws=new WebSocket(tab.webSocketDebuggerUrl);
 ws.addEventListener('message',event=>{
  const response=JSON.parse(event.data);
  if(!response.id)return;
  const item=pending.get(response.id);if(!item)return;
  pending.delete(response.id);
  if(response.error)item.reject(Error(JSON.stringify(response.error)));
  else item.resolve(response.result);
 });
 await new Promise((resolve,reject)=>{ws.addEventListener('open',resolve,{once:true});ws.addEventListener('error',reject,{once:true})});
 await send('Page.enable');await send('Runtime.enable');
 const frame=(await send('Page.getFrameTree')).frameTree.frame.id;
 const html=fs.readFileSync(htmlPath,'utf8');
 async function evaluate(expression){
  const result=await send('Runtime.evaluate',{expression,returnByValue:true});
  if(result.exceptionDetails)throw Error(JSON.stringify(result.exceptionDetails));
  return result.result.value;
 }
 async function state(wanted,step){
  const actual=await evaluate('(()=>{const cards=[...document.querySelectorAll("li.candidate")];return {n:cards.length,visible:cards.filter(x=>!x.hidden).length,shown:document.querySelector("#visible-count").textContent,value:document.querySelector("#source-search").value,overflow:document.documentElement.scrollWidth>window.innerWidth+1,innerWidth:innerWidth,scrollWidth:document.documentElement.scrollWidth,clientWidth:document.documentElement.clientWidth,offenders:[...document.querySelectorAll("*")].filter(x=>x.getBoundingClientRect().right>innerWidth+1).slice(0,8).map(x=>({tag:x.tagName,className:typeof x.className==="string"?x.className:"",right:Math.round(x.getBoundingClientRect().right),scroll:x.scrollWidth,client:x.clientWidth}))};})()');
  if(actual.n!==64||actual.visible!==wanted||Number(actual.shown)!==wanted||actual.overflow)
   throw Error(step+': '+JSON.stringify(actual));
  return actual;
 }
 for(const [width,height,mobile] of [[1280,900,false],[390,844,true]]){
  await send('Emulation.setDeviceMetricsOverride',{width,height,deviceScaleFactor:1,mobile});
  await send('Page.setDocumentContent',{frameId:frame,html});
  for(let i=0;i<50;i++){
   if(await evaluate('document.querySelectorAll("li.candidate").length===64'))break;
   await sleep(50);
  }
  await state(64,'initial');
  await evaluate('document.querySelector("#source-search").focus()');
  await send('Input.insertText',{text:'right shoulder'});
  await sleep(100);
  const filtered=await state(expect.visible_count,'CDP native browser text entry');
  if(filtered.value!=='right shoulder')throw Error('native text did not reach input');
  const shotFilter=await send('Page.captureScreenshot',{format:'png',fromSurface:true,captureBeyondViewport:false});
  const label=mobile?'mobile':'desktop';
  fs.writeFileSync(path.join(path.dirname(htmlPath),'typed-filter-'+label+'.png'),Buffer.from(shotFilter.data,'base64'));
  const rect=await evaluate('(()=>{const r=document.querySelector("#clear-search").getBoundingClientRect();return {x:r.left+r.width/2,y:r.top+r.height/2};})()');
  await send('Input.dispatchMouseEvent',{type:'mousePressed',x:rect.x,y:rect.y,button:'left',clickCount:1});
  await send('Input.dispatchMouseEvent',{type:'mouseReleased',x:rect.x,y:rect.y,button:'left',clickCount:1});
  await sleep(100);
  const restored=await state(64,'native pointer Clear');
  if(restored.value!=='')throw Error('Clear did not empty source search');
  const shotRestored=await send('Page.captureScreenshot',{format:'png',fromSurface:true,captureBeyondViewport:false});
  fs.writeFileSync(path.join(path.dirname(htmlPath),'clear-restored-'+label+'.png'),Buffer.from(shotRestored.data,'base64'));
  console.log('PASS original source Chromium CDP '+width+'x'+height+' typed filter 2/64, pointer Clear 64/64, no horizontal overflow');
 }
}finally{
 try{ws?.close()}catch{}
 chrome.kill('SIGTERM');
 try{fs.rmSync(tmp,{recursive:true,force:true})}catch{}
}

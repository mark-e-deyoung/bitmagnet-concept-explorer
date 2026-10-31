#!/usr/bin/env node
// Synthetic, isolated research-only HTTP fixture for Lighthouse user-flow
// qualification. Never deployed. No Bitmagnet, tracker, DHT, or external I/O.
import http from 'node:http';
import {readFile} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
import path from 'node:path';
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'../..');
const html=await readFile(path.join(root,'web/index.html'));
const port=Number(process.env.UX_FIXTURE_PORT||4175);
const host='127.0.0.1';
const HASH='1111111111111111111111111111111111111111';
const catalog={schema_version:'bce-graph-preview-v1',snapshot_id:'public-ux-fixture-only',
 digest:'a'.repeat(64),authority_status:'pinned-preview-not-normative',
 concepts:[
  {id:'public:creative',label:'Creative work',definition:'Public synthetic authored-work concept',
   lexicalizations:[{language:'en',term:'creative work'},{language:'de',term:'kreatives Werk'},
    {language:'th',term:'งานสร้างสรรค์'}],
   relations:[{predicate:'related',target_id:'public:edition'}]},
  {id:'public:edition',label:'Edition',definition:'Public synthetic edition concept',
   lexicalizations:[{language:'en',term:'edition'},{language:'ja',term:'版'}],relations:[]}
 ]};
const records=[
 {info_hash:HASH,title:'Creative work edition — public synthetic fixture',content_type:'ebook',
  languages:['en','de','th'],seeders:0,leechers:0,adult_metadata:null,
  tracks:['direct','concept'],matched_queries:['creative work','edition']},
 {info_hash:'2222222222222222222222222222222222222222',title:'More research-only evidence',
  content_type:'software',languages:['ja'],seeders:1,leechers:0,adult_metadata:false,
  tracks:['direct'],matched_queries:['creative work']}
];
const observed={
 status:'lexical_candidates_observed',
 concepts:[
  {id:'public:creative',label:'Creative work',distinct_returned_torrents:1},
  {id:'public:edition',label:'Edition',distinct_returned_torrents:1}
 ],
 evidence_edges:[
  {concept_id:'public:creative',info_hash:HASH,surface:'title',matched_term:'Creative work',
   assessment_eligible:false,evidence_role:'candidate_lexical_match'},
  {concept_id:'public:edition',info_hash:HASH,surface:'title',matched_term:'edition',
   assessment_eligible:false,evidence_role:'candidate_lexical_match'}
 ]};
const facets={total_count:172,total_count_is_estimate:true,
 scope:'native_corpus_metadata_aggregate_not_cpe_concepts',
 groups:{
  contentType:[{label:'Ebook',count:54,is_estimate:false},{label:'Software',count:88,is_estimate:true}],
  language:[{label:'English',count:70,is_estimate:true},{label:'German',count:22,is_estimate:false}],
  genre:[{label:'Unclassified',count:14,is_estimate:true}]
 }};
const fixtures={
 '/api/v1/capabilities':{execute:true,mutation:false,cpe_write:false,inference:false,external_web:false},
 '/api/v1/concepts':catalog,
 '/api/v1/sweeps/limits':{remaining_searxng_json_requests:0,max_searxng_json_requests_per_hour:3,
  searxng_cooldown_seconds:60,sweep_deadline_seconds:55,client_timeout_seconds:80,
  max_concurrent_sweeps:1,searxng_sweep_admitted:false},
 '/api/v1/query/execute':{state:'complete',read_only:true,results:records,
  track_counts:{direct:2,concept:1},failures:[],warnings:[],groups:[]},
 '/api/v1/query/plan':{state:'planned',read_only:true,results:[],
  track_counts:{direct:0,concept:0},failures:[],warnings:[]},
 '/api/v1/query/observed-graph':observed,
 '/api/v1/query/facets':facets,
 '/api/v1/sweeps/plan':{state:'planned',bitmagnet_queries:[],searxng_queries:[]},
 '/api/v1/sweeps/execute':{state:'complete',bitmagnet_queries:[],searxng_queries:[]}
};
const posts=new Set(['/api/v1/query/execute','/api/v1/query/plan',
 '/api/v1/query/observed-graph','/api/v1/query/facets',
 '/api/v1/sweeps/plan','/api/v1/sweeps/execute']);
function send(res,code,body,type='application/json; charset=utf-8'){
 const bytes=typeof body==='string'||Buffer.isBuffer(body)?body:JSON.stringify(body);
 const out=Buffer.isBuffer(bytes)?bytes:Buffer.from(bytes);
 res.writeHead(code,{'Content-Type':type,'Content-Length':out.length,
  'Cache-Control':'no-store','X-Content-Type-Options':'nosniff',
  'X-Robots-Tag':'noindex, nofollow',
  'Content-Security-Policy':"default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'"});
 res.end(out);
}
const server=http.createServer(async(req,res)=>{
 try{
  const url=new URL(req.url, 'http://127.0.0.1');
  if(url.pathname==='/'&&req.method==='GET'){send(res,200,html,'text/html; charset=utf-8');return;}
  if(url.pathname==='/healthz'&&req.method==='GET'){send(res,200,{fixture:true,isolated:true});return;}
  if(url.pathname==='/robots.txt'&&req.method==='GET'){send(res,200,'User-agent: *\nDisallow: /\n','text/plain');return;}
  if(url.pathname==='/favicon.ico'&&req.method==='GET'){send(res,204,'','image/x-icon');return;}
  if(!Object.hasOwn(fixtures,url.pathname)){send(res,404,{error:'synthetic fixture route not provided'});return;}
  if(req.method==='POST'&&!posts.has(url.pathname)){send(res,405,{error:'read-only fixture'});return;}
  if(req.method!=='POST'&&req.method!=='GET'){send(res,405,{error:'read-only fixture'});return;}
  if(req.method==='POST'){
   let n=0;
   for await(const chunk of req){
    n+=chunk.length;
    if(n>131072){send(res,413,{error:'fixture request exceeds cap'});return;}
   }
  }
  send(res,200,fixtures[url.pathname]);
 }catch(err){send(res,500,{error:'synthetic fixture internal error'});}
});
server.requestTimeout=15000;
server.listen(port,host,()=>process.stdout.write('UX_FIXTURE_READY=http://'+host+':'+port+'\n'));

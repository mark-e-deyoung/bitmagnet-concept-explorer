// Command inspect provides read-only original-source CPE research projections
// for human inspection, operator JSON and agent capability discovery.
// No ontology mutation, remote service, character media or rights escalation.
package main
import (
 "bufio"
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "errors"
 "flag"
 "fmt"
 "html/template"
 "io"
 "os"
 "strings"
)
const pinned="7294644f1fc26490f8a0aed415afc01b8278294e4373169e1407986c56cb0961"
const commit="1d5ca33372491509805a64cd0425a5164d3108fa"
const blob="08984be691ebd69f92a7a271217612f0056cea37"
type Candidate struct{
 ID string `json:"id"`
 Line int `json:"line"`
 Evidence string `json:"evidence"`
 Motion []string `json:"motion_terms"`
 Actor []string `json:"actor_terms"`
 Timing []string `json:"timing_terms"`
 Status string `json:"status"`
 Measurement string `json:"measurement_status"`
}
type SourceReport struct{
 Schema string `json:"schema"`
 SourceID string `json:"source_id"`
 Work string `json:"work"`
 URL string `json:"source_url"`
 Commit string `json:"git_commit"`
 Blob string `json:"git_blob"`
 SHA string `json:"source_sha256"`
 Bytes int `json:"input_bytes"`
 Truncated bool `json:"truncated"`
 Candidates []Candidate `json:"candidates"`
}
type EvidenceView struct{
 Index int `json:"index"`
 CandidateID string `json:"candidate_id"`
 Line int `json:"line"`
 Evidence string `json:"evidence"`
 ContextStart int `json:"context_start_line"`
 ContextEnd int `json:"context_end_line"`
 Context string `json:"context"`
 MotionTerms string `json:"motion_terms_text"`
 ActorTerms string `json:"actor_or_direction_terms_text"`
 TimingTerms string `json:"timing_terms_text"`
 RoleInterpretation string `json:"role_interpretation_status"`
 ContextStatus string `json:"context_status"`
 SearchText string `json:"search_text"`
 MatchOrigin string `json:"match_origin"`
}
type Inspection struct{
 Schema string `json:"schema"`
 SourceID string `json:"source_id"`
 Work string `json:"work"`
 SourceURL string `json:"source_url"`
 SourceSHA string `json:"source_sha256"`
 Count int `json:"candidate_count"`
 VisibleCount int `json:"visible_count"`
 Query string `json:"query"`
 Status string `json:"status"`
 ClaimsPhysicalMotion bool `json:"claims_physical_motion"`
 ClaimsChoreographyAcceptance bool `json:"claims_choreography_acceptance"`
 Candidates []Candidate `json:"candidates"`
 Views []EvidenceView `json:"views"`
}
type Capabilities struct{
 Schema string `json:"schema"`
 HumanUI string `json:"human_ui"`
 Automation string `json:"automation"`
 Agent string `json:"agent"`
 Modes []string `json:"modes"`
 Holds []string `json:"holds"`
}
func discover()Capabilities{
 return Capabilities{Schema:"scp.choreography-inspection-capability.v0",
 HumanUI:"OFFLINE_READ_ONLY_HTML_NO_DEVICE_HIL",Automation:"SOURCE_PINNED_CLI_JSON",
 Agent:"DISCOVERABLE_READ_ONLY_CAPABILITIES",
 Modes:[]string{"-capabilities","-format=json -source= -report= -query=","-format=html -source= -report= -query="},
 Holds:[]string{"research-only","no-native-browser-qa","no-editing","no-accepted-movement","no-CPE-promotion","no-private-character-ingress"}}
}
func contextWindow(lines []string,line int)(int,int,string){
 start,end:=line-1,line-1
 for n:=0;n<2&&start>0&&strings.TrimSpace(lines[start-1])!="";n++{start--}
 for n:=0;n<2&&end+1<len(lines)&&strings.TrimSpace(lines[end+1])!="";n++{end++}
 return start+1,end+1,strings.Join(lines[start:end+1],"\n")
}
func roleCue(a []string)string{
 for _,v:=range a{
  switch strings.ToLower(v){
  case "opposite","opposites","side","all","foot","feet","hands","hand","files","corners":
   return "ACTOR_OR_DIRECTION_AMBIGUOUS_NOT_RESOLVED"
  }
 }
 return "SOURCE_LEXICAL_MATCH_ONLY_NOT_VALIDATED_ACTOR"
}
func anchored(source []byte,r SourceReport)(Inspection,error){
 if len(source)==0||len(source)>2<<20{return Inspection{},errors.New("source bounds")}
 digest:=sha256.Sum256(source);got:=hex.EncodeToString(digest[:])
 if got!=pinned||r.SHA!=got||r.Bytes!=len(source)||r.Commit!=commit||r.Blob!=blob||r.SourceID!="pg:12926"||r.Schema!="cpe.motion-text-candidates.v0"{
  return Inspection{},errors.New("source or candidate identity mismatch")
 }
 requiredURL:="https://github.com/GITenberg/The-Morris-Book-Part-1--13-A-History-of-Morris-Dancing-With-a-Description-of-Eleven-Dances-as__12926/blob/"+commit+"/12926.txt"
 if r.Work!="The Morris Book, Part 1"||r.URL!=requiredURL {
  return Inspection{},errors.New("original work/edition URL mismatch")
 }
 if len(r.Candidates)>10000{return Inspection{},errors.New("too many candidates")}
 var lines []string
 scanner:=bufio.NewScanner(strings.NewReader(string(source)))
 scanner.Buffer(make([]byte,4096),2<<20)
 for scanner.Scan(){lines=append(lines,strings.TrimRight(scanner.Text(),"\r"))}
 if e:=scanner.Err();e!=nil{return Inspection{},e}
 seen:=map[string]bool{}
 views:=[]EvidenceView{}
 for i,c:=range r.Candidates{
  if c.ID==""||seen[c.ID]||c.Line<1||c.Line>len(lines)||c.Evidence==""||c.Status!="SOURCE_TEXT_RESEARCH_CANDIDATE"||c.Measurement!="UNMEASURED_TEXT_INSTRUCTION"{
   return Inspection{},errors.New("invalid candidate provenance or unsupported claim")
  }
  original:=strings.TrimSpace(lines[c.Line-1])
  if !strings.HasPrefix(original,c.Evidence){return Inspection{},errors.New("source anchor/evidence mismatch")}
  identity:=sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%s",r.SourceID,c.Line,original)))
  if c.ID!="pg12926-motion-"+hex.EncodeToString(identity[:8]){return Inspection{},errors.New("forged candidate identifier")}
  seen[c.ID]=true
  a,b,context:=contextWindow(lines,c.Line)
  views=append(views,EvidenceView{Index:i,CandidateID:c.ID,Line:c.Line,Evidence:c.Evidence,
   ContextStart:a,ContextEnd:b,Context:context,MotionTerms:strings.Join(c.Motion,", "),
   ActorTerms:strings.Join(c.Actor,", "),TimingTerms:strings.Join(c.Timing,", "),
   RoleInterpretation:roleCue(c.Actor),ContextStatus:"ADJACENT_SOURCE_LINES_NOT_VERIFIED_FIGURE",
   SearchText:strings.ToLower(strings.Join([]string{c.ID,c.Evidence,context,strings.Join(c.Motion," "),strings.Join(c.Actor," "),strings.Join(c.Timing," ")}," "))})
 }
 return Inspection{Schema:"scp.choreography-inspection-readmodel.v0",SourceID:r.SourceID,Work:r.Work,SourceURL:r.URL,SourceSHA:got,
 Count:len(r.Candidates),VisibleCount:len(r.Candidates),Status:"UNVALIDATED_SOURCE_TEXT_CANDIDATES",Candidates:r.Candidates,Views:views},nil
}
func filterInspection(v Inspection,query string)(Inspection,error){
 q:=strings.ToLower(strings.TrimSpace(query))
 if len([]rune(q))>80{return Inspection{},errors.New("query exceeds 80 characters")}
 result:=v
 result.Query=q
 result.Views=[]EvidenceView{}
 result.Candidates=[]Candidate{}
 for _,ev:=range v.Views{
  if strings.Contains(ev.SearchText,q){
   switch{
   case q=="":ev.MatchOrigin="UNFILTERED"
   case strings.Contains(strings.ToLower(ev.Evidence),q):ev.MatchOrigin="ORIGINAL_SOURCE_LINE"
   case strings.Contains(strings.ToLower(ev.Context),q):ev.MatchOrigin="ADJACENT_SOURCE_CONTEXT"
   default:ev.MatchOrigin="SOURCE_METADATA_ONLY"
   }
   result.Views=append(result.Views,ev)
   result.Candidates=append(result.Candidates,v.Candidates[ev.Index])
  }
 }
 result.VisibleCount=len(result.Views)
 return result,nil
}
const page=`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Historical choreography — source review</title>
<style>
:root{color-scheme:light dark}*{box-sizing:border-box}
body{font:1rem/1.48 system-ui,sans-serif;margin:0;color:#16212c;background:#fff}
main{max-width:70rem;padding:clamp(1rem,3vw,2rem);margin:auto;min-width:0}
h1{font-size:clamp(1.4rem,3.8vw,2rem);line-height:1.2;margin:0 0 1rem}
p{margin:.65rem 0}.caution{border-left:4px solid #8b5e00;padding:.7rem 1rem;background:#fcf9ee}
.meta{font-size:.91rem;overflow-wrap:anywhere}.meta code{font-size:.85rem}
ol.results{list-style:none;padding:0;margin:1.25rem 0;display:grid;gap:.7rem}
li.candidate{margin:0;border:1px solid #9aa9b5;border-radius:.35rem;min-width:0}
details{min-width:0}summary{display:grid;grid-template-columns:auto minmax(0,1fr);gap:.6rem;align-items:start;cursor:pointer;padding:.8rem}
summary:hover{background:#eff5f9}summary:focus-visible{outline:3px solid #995400;outline-offset:2px}
.line{white-space:nowrap;font-weight:700}.preview{min-width:0;overflow-wrap:anywhere}
.details{padding:.1rem 1rem 1rem;border-top:1px solid #afbac4}
.source-context{white-space:pre-wrap;overflow-wrap:anywhere;line-height:1.6;background:#f1f4f6;padding:.8rem;border-left:3px solid #6c8091;margin:.5rem 0}
.small{font-size:.89rem}.small code{overflow-wrap:anywhere}
.match-origin{display:block;color:#4d5865;font-size:.8rem;font-weight:600;margin-top:.22rem}
.match-origin[hidden]{display:none}
@media(prefers-color-scheme:dark){.match-origin{color:#bdc9d7}}
.search{display:flex;gap:.5rem;align-items:center;flex-wrap:wrap;margin:1rem 0}
.search label{font-weight:600}.search input{flex:1 1 15rem;min-width:0;max-width:30rem;padding:.55rem;font:inherit;background:transparent;color:inherit;border:1px solid #73869a}
.search button{padding:.55rem .85rem;border:1px solid #6c8091;background:transparent;color:inherit;cursor:pointer}
.search button:focus-visible,.search input:focus-visible{outline:3px solid #995400}
li.candidate[hidden]{display:none!important}
@media(max-width:540px){main{padding:1rem}summary{grid-template-columns:1fr;gap:.3rem}.details{padding:.5rem .75rem 1rem}}
@media(prefers-color-scheme:dark){
 body{background:#151c25;color:#eef3f9}
 .caution{background:#40341d}li.candidate{border-color:#69788b}
 summary:hover{background:#27364a}.details{border-top-color:#69788b}
 .source-context{background:#263341;border-color:#9aabc1}
}
</style></head><body><main>
<h1>Historical choreography: source review</h1>
<p class="caution" role="status">Unreviewed textual matches, not independent dance figures. No accepted actor interpretation, motion measurement, physical contact or CPE concept promotion.</p>
<p class="meta"><strong>Original work:</strong> <cite>{{.Work}}</cite> · <a href="{{.SourceURL}}" rel="noopener noreferrer">View pinned original source</a></p>
<p class="meta"><strong>Source:</strong> <code>{{.SourceID}}</code> · <strong>SHA-256:</strong> <code>{{.SourceSHA}}</code></p>
<p><strong>Occurrences:</strong> {{.Count}} · <strong>Scientific state:</strong> unvalidated source-text candidates.</p>
<p id="guide">Select a source occurrence to inspect adjacent original lines. These windows provide context, <strong>not</strong> verified complete figures.</p>
<form class="search" role="search" onsubmit="return false">
<label for="source-search">Filter source evidence</label><input id="source-search" type="search" maxlength="80" autocomplete="off" value="{{.Query}}" placeholder="e.g. right shoulder">
<button id="clear-search" type="button">Clear</button></form>
<p aria-live="polite" aria-atomic="true">Showing <output id="visible-count">{{.VisibleCount}}</output> of {{.Count}} source matches</p>
<ol class="results" aria-label="Source candidate occurrences">
{{range .Views}}<li class="candidate" data-search="{{.SearchText}}"><details {{if eq .Index 0}}open{{end}}>
<summary aria-describedby="guide"><span class="line">Line {{.Line}}</span><span class="preview">{{.Evidence}}<span class="match-origin" hidden></span></span></summary>
<div class="details"><p class="small"><strong>Original context:</strong> lines {{.ContextStart}}–{{.ContextEnd}} (unreviewed)</p>
<blockquote class="source-context">{{.Context}}</blockquote>
<p class="small"><strong>Movement word matches:</strong> {{.MotionTerms}} · <strong>Actor/direction word matches:</strong> {{.ActorTerms}} · <strong>Timing word matches:</strong> {{.TimingTerms}}</p>
<p class="small"><strong>Interpretation:</strong> {{.RoleInterpretation}} · <strong>Context status:</strong> {{.ContextStatus}}</p>
<p class="small"><strong>Candidate ID:</strong> <code>{{.CandidateID}}</code></p></div></details></li>{{end}}
</ol>
<script>
(()=>{"use strict";
 const input=document.getElementById("source-search");
 const output=document.getElementById("visible-count");
 const cards=Array.from(document.querySelectorAll("li.candidate"));
 function apply(){
  const query=input.value.trim().toLowerCase();let shown=0;
  for(const card of cards){
   const keep=card.dataset.search.includes(query);card.hidden=!keep;
   const basis=card.querySelector(".match-origin");
   if(basis){
    const exact=card.querySelector(".preview").childNodes[0]?.textContent.toLowerCase()||"";
    const context=card.querySelector(".source-context")?.textContent.toLowerCase()||"";
    basis.hidden=!keep||!query;
    basis.textContent=!query?"":exact.includes(query)?"Match in original line":context.includes(query)?"Match in adjacent source context":"Match in source metadata";
   }
   if(keep)shown++;
  }
  output.textContent=String(shown);
 }
 input.addEventListener("input",apply);
 document.getElementById("clear-search").addEventListener("click",()=>{input.value="";apply();input.focus();});
 apply();
})();
</script></main></body></html>`

func renderHTML(w io.Writer,v Inspection)error{
 t,err:=template.New("inspect").Parse(page);if err!=nil{return err}
 return t.Execute(w,v)
}
func loadReport(r io.Reader)(SourceReport,error){
 d:=json.NewDecoder(io.LimitReader(r,4<<20));d.DisallowUnknownFields()
 var p SourceReport;if err:=d.Decode(&p);err!=nil{return SourceReport{},err}
 var trailing any;if err:=d.Decode(&trailing);err!=io.EOF{return SourceReport{},errors.New("trailing report JSON")}
 return p,nil
}
func run(args []string,w io.Writer)error{
 fs:=flag.NewFlagSet("inspect",flag.ContinueOnError);fs.SetOutput(io.Discard)
 cap:=fs.Bool("capabilities",false,"discover read-only operations and holds")
 format:=fs.String("format","json","json or html")
 sourcePath:=fs.String("source","","pinned text bytes")
 reportPath:=fs.String("report","","source-bound candidate JSON")
 query:=fs.String("query","","read-only source filter")
 if err:=fs.Parse(args);err!=nil{return err}
 if fs.NArg()!=0{return errors.New("unexpected positional arguments")}
 enc:=json.NewEncoder(w);enc.SetIndent("","  ")
 if *cap{if *sourcePath!=""||*reportPath!=""||*query!=""{return errors.New("capabilities are input-free")};return enc.Encode(discover())}
 if *sourcePath==""||*reportPath==""{return errors.New("source and report are required")}
 if *format!="json"&&*format!="html"{return errors.New("unsupported format")}
 src,err:=os.ReadFile(*sourcePath);if err!=nil{return err}
 f,err:=os.Open(*reportPath);if err!=nil{return err};defer f.Close()
 report,err:=loadReport(f);if err!=nil{return err}
 inspected,err:=anchored(src,report);if err!=nil{return err}
 filtered,err:=filterInspection(inspected,*query);if err!=nil{return err}
 if *format=="html"{inspected.Query=filtered.Query;inspected.VisibleCount=filtered.VisibleCount;return renderHTML(w,inspected)}
 return enc.Encode(filtered)
}
func main(){if err:=run(os.Args[1:],os.Stdout);err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}}

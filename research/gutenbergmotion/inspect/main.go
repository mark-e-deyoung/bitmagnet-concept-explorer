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
type Inspection struct{
 Schema string `json:"schema"`
 SourceID string `json:"source_id"`
 SourceSHA string `json:"source_sha256"`
 Count int `json:"candidate_count"`
 Status string `json:"status"`
 ClaimsPhysicalMotion bool `json:"claims_physical_motion"`
 ClaimsChoreographyAcceptance bool `json:"claims_choreography_acceptance"`
 Candidates []Candidate `json:"candidates"`
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
 Modes:[]string{"-capabilities","-format=json -source= -report=","-format=html -source= -report="},
 Holds:[]string{"research-only","no-native-browser-qa","no-editing","no-accepted-movement","no-CPE-promotion","no-private-character-ingress"}}
}
func anchored(source []byte,r SourceReport)(Inspection,error){
 if len(source)==0||len(source)>2<<20{return Inspection{},errors.New("source bounds")}
 digest:=sha256.Sum256(source);got:=hex.EncodeToString(digest[:])
 if got!=pinned||r.SHA!=got||r.Bytes!=len(source)||r.Commit!=commit||r.Blob!=blob||r.SourceID!="pg:12926"||r.Schema!="cpe.motion-text-candidates.v0"{
  return Inspection{},errors.New("source or candidate identity mismatch")
 }
 if !strings.HasPrefix(r.URL,"https://github.com/GITenberg/The-Morris-Book-Part-1--13-A-History-of-Morris-Dancing-With-a-Description-of-Eleven-Dances-as__12926/blob/"){
  return Inspection{},errors.New("unexpected original source link")
 }
 if len(r.Candidates)>10000{return Inspection{},errors.New("too many candidates")}
 var lines []string
 scanner:=bufio.NewScanner(strings.NewReader(string(source)))
 scanner.Buffer(make([]byte,4096),2<<20)
 for scanner.Scan(){lines=append(lines,strings.TrimRight(scanner.Text(),"\r"))}
 if e:=scanner.Err();e!=nil{return Inspection{},e}
 seen:=map[string]bool{}
 for _,c:=range r.Candidates{
  if c.ID==""||seen[c.ID]||c.Line<1||c.Line>len(lines)||c.Evidence==""||c.Status!="SOURCE_TEXT_RESEARCH_CANDIDATE"||c.Measurement!="UNMEASURED_TEXT_INSTRUCTION"{
   return Inspection{},errors.New("invalid candidate provenance or unsupported claim")
  }
  if !strings.HasPrefix(strings.TrimSpace(lines[c.Line-1]),c.Evidence){return Inspection{},errors.New("source anchor/evidence mismatch")}
  seen[c.ID]=true
 }
 return Inspection{Schema:"scp.choreography-inspection-readmodel.v0",SourceID:r.SourceID,SourceSHA:got,
 Count:len(r.Candidates),Status:"UNVALIDATED_SOURCE_TEXT_CANDIDATES",Candidates:r.Candidates},nil
}
const page=`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Historical choreography research evidence</title>
<style>
body{font:1rem/1.5 system-ui,sans-serif;max-width:74rem;padding:1rem;margin:auto;color:#16212c;background:#fff}
a{color:#064e91}a:focus-visible{outline:3px solid #a34800}table{border-collapse:collapse;width:100%}
th,td{padding:.6rem;border:1px solid #999;text-align:left;vertical-align:top}th{background:#ebf1f5}
code{overflow-wrap:anywhere}main{min-width:0}.scroll{overflow-x:auto}.note{border-inline-start:4px solid #8b5e00;padding:.5rem 1rem;background:#fcf9ee}
@media(prefers-color-scheme:dark){body{background:#151c25;color:#eef3f9}a{color:#92c5ff}th{background:#243444}.note{background:#40341d}}
</style></head><body>
<main><h1>Historical choreography: source evidence</h1>
<p class="note" role="status">Research candidates only. No verified complete dance figure, true human motion, validated roles or accepted CPE concepts.</p>
<p><strong>Source:</strong> <code>{{.SourceID}}</code> · <strong>SHA-256:</strong> <code>{{.SourceSHA}}</code></p>
<p><strong>Candidate occurrences:</strong> {{.Count}} · <strong>Scientific status:</strong> {{.Status}}</p>
<div class="scroll"><table><caption>Original-source candidate lines (not verified movements)</caption>
<thead><tr><th scope="col">Original line</th><th scope="col">Source text</th><th scope="col">Candidate status</th></tr></thead>
<tbody>{{range .Candidates}}<tr><td><code>{{.Line}}</code></td><td>{{.Evidence}}</td><td>{{.Measurement}}</td></tr>{{end}}</tbody></table></div>
</main></body></html>`
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
 if err:=fs.Parse(args);err!=nil{return err}
 if fs.NArg()!=0{return errors.New("unexpected positional arguments")}
 enc:=json.NewEncoder(w);enc.SetIndent("","  ")
 if *cap{if *sourcePath!=""||*reportPath!=""{return errors.New("capabilities are input-free")};return enc.Encode(discover())}
 if *sourcePath==""||*reportPath==""{return errors.New("source and report are required")}
 if *format!="json"&&*format!="html"{return errors.New("unsupported format")}
 src,err:=os.ReadFile(*sourcePath);if err!=nil{return err}
 f,err:=os.Open(*reportPath);if err!=nil{return err};defer f.Close()
 report,err:=loadReport(f);if err!=nil{return err}
 inspected,err:=anchored(src,report);if err!=nil{return err}
 if *format=="html"{return renderHTML(w,inspected)}
 return enc.Encode(inspected)
}
func main(){if err:=run(os.Args[1:],os.Stdout);err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}}

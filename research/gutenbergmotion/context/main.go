// Command context expands source-pinned research candidates into bounded source text.
// Context is NOT a verified dance figure, pose, force, score or CPE fact.
package main

import (
 "bufio"
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "errors"
 "flag"
 "fmt"
 "io"
 "os"
 "regexp"
 "strings"
)
const sourceSHA="7294644f1fc26490f8a0aed415afc01b8278294e4373169e1407986c56cb0961"
const sourceCommit="1d5ca33372491509805a64cd0425a5164d3108fa"
const sourceBlob="08984be691ebd69f92a7a271217612f0056cea37"
const maxSource=2<<20
type Candidate struct {
 ID string `json:"id"`
 Line int `json:"line"`
 Evidence string `json:"evidence"`
 Actors []string `json:"actor_terms"`
 Status string `json:"status"`
 Measurement string `json:"measurement_status"`
}
type Report struct {
 Schema string `json:"schema"`
 SourceID string `json:"source_id"`
 GitCommit string `json:"git_commit"`
 GitBlob string `json:"git_blob"`
 SourceSHA256 string `json:"source_sha256"`
 InputBytes int `json:"input_bytes"`
 Truncated bool `json:"truncated"`
 Candidates []Candidate `json:"candidates"`
}
type Window struct {
 CandidateID string `json:"candidate_id"`
 AnchorLine int `json:"anchor_line"`
 StartLine int `json:"start_line"`
 EndLine int `json:"end_line"`
 SourceLines []string `json:"source_lines"`
 RoleStatus string `json:"role_status"`
 Status string `json:"status"`
}
type Result struct {
 Schema string `json:"schema"`
 SourceID string `json:"source_id"`
 SourceSHA256 string `json:"source_sha256"`
 CandidateCount int `json:"candidate_count"`
 SourceCandidateTruncated bool `json:"source_candidate_truncated"`
 CompleteFigureClaim bool `json:"complete_figure_claim"`
 MotionMeasurementClaim bool `json:"motion_measurement_claim"`
 Windows []Window `json:"windows"`
}
var numbered=regexp.MustCompile("(?i)\\bnos?\\.?\\s*[0-9]+\\b")
func roleStatus(c Candidate) string{
 for _,v:=range c.Actors {
  switch v {case "opposite","opposites","side","all","foot","feet","hand","hands","files","corners":
   return "AMBIGUOUS_SOURCE_ROLE"}
 }
 if numbered.MatchString(strings.Join(c.Actors," ")){return "NUMBERED_ROLE_UNREVIEWED"}
 return "UNRESOLVED_SOURCE_ROLE"
}
func linesOf(src []byte)([]string,error){
 sc:=bufio.NewScanner(strings.NewReader(string(src)))
 sc.Buffer(make([]byte,4096),maxSource)
 var out []string
 for sc.Scan(){out=append(out,strings.TrimSuffix(sc.Text(),"\r"))}
 return out,sc.Err()
}
func Enrich(src []byte,r Report,requirePin bool)(Result,error){
 if len(src)==0||len(src)>maxSource{return Result{},errors.New("source size rejected")}
 h:=sha256.Sum256(src);sum:=hex.EncodeToString(h[:])
 if requirePin&&sum!=sourceSHA{return Result{},errors.New("source bytes do not match pinned original")}
 if r.Schema!="cpe.motion-text-candidates.v0"||r.SourceID!="pg:12926"||r.GitCommit!=sourceCommit||r.GitBlob!=sourceBlob||r.SourceSHA256!=sum||r.InputBytes!=len(src) {
  return Result{},errors.New("candidate report source identity mismatch")
 }
 if len(r.Candidates)>10000{return Result{},errors.New("too many candidates")}
 lines,err:=linesOf(src);if err!=nil{return Result{},err}
 result:=Result{Schema:"cpe.motion-text-context.v1",SourceID:r.SourceID,SourceSHA256:sum,CandidateCount:len(r.Candidates),SourceCandidateTruncated:r.Truncated,Windows:[]Window{}}
 seen:=map[string]bool{}
 for _,c:=range r.Candidates {
  if c.ID==""||seen[c.ID]||c.Line<1||c.Line>len(lines)||c.Evidence=="" {return Result{},errors.New("duplicate ID or invalid source anchor")}
  seen[c.ID]=true
  if !strings.HasPrefix(strings.TrimSpace(lines[c.Line-1]),c.Evidence)||c.Measurement!="UNMEASURED_TEXT_INSTRUCTION"||c.Status!="SOURCE_TEXT_RESEARCH_CANDIDATE" {
   return Result{},fmt.Errorf("candidate %q source or claim forged",c.ID)
  }
  start,end:=c.Line-1,c.Line-1
  for n:=0;n<2&&start>0&&strings.TrimSpace(lines[start-1])!="";n++{start--}
  for n:=0;n<2&&end+1<len(lines)&&strings.TrimSpace(lines[end+1])!="";n++{end++}
  result.Windows=append(result.Windows,Window{CandidateID:c.ID,AnchorLine:c.Line,StartLine:start+1,EndLine:end+1,
   SourceLines:append([]string(nil),lines[start:end+1]...),RoleStatus:roleStatus(c),Status:"UNREVIEWED_CONTEXT_WINDOW_NOT_VALIDATED_FIGURE"})
 }
 return result,nil
}
func run(args []string,w io.Writer) error{
 fs:=flag.NewFlagSet("context",flag.ContinueOnError);fs.SetOutput(io.Discard)
 srcName:=fs.String("source","","original text")
 jsonName:=fs.String("candidates","","original candidate JSON")
 if err:=fs.Parse(args);err!=nil{return err}
 if *srcName==""||*jsonName==""||fs.NArg()!=0{return errors.New("source and candidates required")}
 src,err:=os.ReadFile(*srcName);if err!=nil{return err}
 f,err:=os.Open(*jsonName);if err!=nil{return err};defer f.Close()
 var report struct{
  Schema string `json:"schema"`
  SourceID string `json:"source_id"`
  Work string `json:"work"`
  SourceURL string `json:"source_url"`
  GitCommit string `json:"git_commit"`
  GitBlob string `json:"git_blob"`
  SourceSHA256 string `json:"source_sha256"`
  InputBytes int `json:"input_bytes"`
  Truncated bool `json:"truncated"`
  Candidates []struct{
   ID string `json:"id"`
   Line int `json:"line"`
   Evidence string `json:"evidence"`
   Motions []string `json:"motion_terms"`
   Actors []string `json:"actor_terms"`
   Timing []string `json:"timing_terms"`
   Status string `json:"status"`
   Measurement string `json:"measurement_status"`
  } `json:"candidates"`
 }
 d:=json.NewDecoder(io.LimitReader(f,4<<20));d.DisallowUnknownFields()
 if err:=d.Decode(&report);err!=nil{return err}
 var extra any;if err:=d.Decode(&extra);err!=io.EOF{return errors.New("trailing candidate JSON")}
 r:=Report{Schema:report.Schema,SourceID:report.SourceID,GitCommit:report.GitCommit,GitBlob:report.GitBlob,SourceSHA256:report.SourceSHA256,InputBytes:report.InputBytes,Truncated:report.Truncated}
 for _,c:=range report.Candidates {r.Candidates=append(r.Candidates,Candidate{ID:c.ID,Line:c.Line,Evidence:c.Evidence,Actors:c.Actors,Status:c.Status,Measurement:c.Measurement})}
 o,err:=Enrich(src,r,true);if err!=nil{return err}
 enc:=json.NewEncoder(w);enc.SetIndent("","  ");return enc.Encode(o)
}
func main(){if err:=run(os.Args[1:],os.Stdout);err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}}

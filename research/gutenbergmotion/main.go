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
  "sort"
  "strings"
)

const sourceID = "pg:12926"
const gitCommit = "1d5ca33372491509805a64cd0425a5164d3108fa"
const gitBlob = "08984be691ebd69f92a7a271217612f0056cea37"
const sourceSHA256 = "7294644f1fc26490f8a0aed415afc01b8278294e4373169e1407986c56cb0961"
const maxSourceBytes = 2 << 20

type Candidate struct {
  ID string `json:"id"`
  Line int `json:"line"`
  Evidence string `json:"evidence"`
  Motions []string `json:"motion_terms"`
  Actors []string `json:"actor_terms"`
  Timing []string `json:"timing_terms"`
  Status string `json:"status"`
  Measurement string `json:"measurement_status"`
}
type Report struct {
  Schema string `json:"schema"`
  SourceID string `json:"source_id"`
  Work string `json:"work"`
  SourceURL string `json:"source_url"`
  GitCommit string `json:"git_commit"`
  GitBlob string `json:"git_blob"`
  SourceSHA256 string `json:"source_sha256"`
  InputBytes int `json:"input_bytes"`
  Truncated bool `json:"truncated"`
  Candidates []Candidate `json:"candidates"`
}
var motion = regexp.MustCompile(`(?i)\b(advanc(?:e|es|ing)|retir(?:e|es|ing)|cross.over|back.to.back|chain|jump(?:s|ing)?|stamp(?:s|ing)?|step(?:s|ping)?|strik(?:e|es|ing)|tap(?:s|ping)?|turn(?:s|ing)?|swing(?:s|ing)?|pass(?:es|ing)?|clap(?:s|ping)?|ring|fac(?:e|es|ing)|return(?:s|ing)?|follow(?:s|ing)?|hand.striking|mov(?:e|es|ing))\b`)
var actor = regexp.MustCompile(`(?i)\b(nos?\.?\s*\d+|partners?|dancers?|side|all|feet|foot|hands?|corners?|opposites?|files?|leader)\b`)
var timing = regexp.MustCompile(`(?i)\b(bars?\s*\d+(?:\s*(?:to|through|-)\s*\d+)?|\d+(?:st|nd|rd|th)?\s+(?:half-)?bars?|half.bar|beats?|music\s+[abc]\d?)\b`)

func unique(in []string) []string {
  seen:=map[string]bool{};out:=[]string{}
  for _,v:=range in {v=strings.ToLower(strings.TrimSpace(v));if v!=""&&!seen[v]{seen[v]=true;out=append(out,v)}}
  sort.Strings(out);return out
}
func Mine(content []byte,limit int)(Report,error){
  if len(content)==0||len(content)>maxSourceBytes{return Report{},fmt.Errorf("invalid source size %d",len(content))}
  if limit<1||limit>10000{return Report{},errors.New("invalid candidate limit")}
  digest:=sha256.Sum256(content)
  r:=Report{Schema:"cpe.motion-text-candidates.v0",SourceID:sourceID,Work:"The Morris Book, Part 1",
    SourceURL:"https://github.com/GITenberg/The-Morris-Book-Part-1--13-A-History-of-Morris-Dancing-With-a-Description-of-Eleven-Dances-as__12926/blob/"+gitCommit+"/12926.txt",
    GitCommit:gitCommit,GitBlob:gitBlob,SourceSHA256:hex.EncodeToString(digest[:]),
    InputBytes:len(content),Candidates:[]Candidate{}}
  sc:=bufio.NewScanner(strings.NewReader(string(content)))
  sc.Buffer(make([]byte,0,65536),maxSourceBytes)
  for line:=1;sc.Scan();line++{
    s:=strings.TrimSpace(sc.Text());if s==""{continue}
    m:=unique(motion.FindAllString(s,-1));a:=unique(actor.FindAllString(s,-1));t:=unique(timing.FindAllString(s,-1))
    if len(m)==0||len(a)==0||len(t)==0{continue}
    ev:=s;if len(ev)>280{ev=ev[:280]}
    h:=sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%s",sourceID,line,s)))
    r.Candidates=append(r.Candidates,Candidate{ID:"pg12926-motion-"+hex.EncodeToString(h[:8]),Line:line,
      Evidence:ev,Motions:m,Actors:a,Timing:t,Status:"SOURCE_TEXT_RESEARCH_CANDIDATE",Measurement:"UNMEASURED_TEXT_INSTRUCTION"})
    if len(r.Candidates)>=limit{r.Truncated=true;break}
  }
  if err:=sc.Err();err!=nil{return Report{},err}
  return r,nil
}
func run(args []string,out io.Writer)error{
  flags:=flag.NewFlagSet("gutenberg-motion",flag.ContinueOnError)
  flags.SetOutput(io.Discard)
  input:=flags.String("input","","path to pinned original text")
  limit:=flags.Int("limit",600,"candidate cap")
  if err:=flags.Parse(args);err!=nil{return err}
  if *input==""{return errors.New("-input required")}
  b,err:=os.ReadFile(*input);if err!=nil{return err}
  r,err:=Mine(b,*limit);if err!=nil{return err}
  if r.SourceSHA256!=sourceSHA256{return fmt.Errorf("source bytes do not match pinned original: got %s",r.SourceSHA256)}
  enc:=json.NewEncoder(out);enc.SetIndent("","  ");return enc.Encode(r)
}
func main(){if err:=run(os.Args[1:],os.Stdout);err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}}

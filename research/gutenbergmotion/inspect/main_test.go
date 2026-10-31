package main
import(
 "bytes"
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "strings"
 "testing"
)
func synthetic()([]byte,SourceReport){
 b:=[]byte("Bar 4. No. 1 advances and <script>alert('bad')</script>.\n")
 h:=sha256.Sum256(b)
 return b,SourceReport{Schema:"cpe.motion-text-candidates.v0",SourceID:"pg:12926",
 Commit:commit,Blob:blob,SHA:hex.EncodeToString(h[:]),Bytes:len(b),
 URL:"https://github.com/GITenberg/The-Morris-Book-Part-1--13-A-History-of-Morris-Dancing-With-a-Description-of-Eleven-Dances-as__12926/blob/"+commit+"/12926.txt",
 Candidates:[]Candidate{{ID:"test",Line:1,Evidence:"Bar 4. No. 1 advances and <script>alert('bad')</script>.",Status:"SOURCE_TEXT_RESEARCH_CANDIDATE",Measurement:"UNMEASURED_TEXT_INSTRUCTION"}}}
}
func TestHumanAutomationAgentSharedReadModel(t *testing.T){
 b,r:=synthetic()
 if _,e:=anchored(b,r);e==nil{t.Fatal("unpinned fake source accepted")}
 if discover().Agent!="DISCOVERABLE_READ_ONLY_CAPABILITIES"{t.Fatal("missing agent contract")}
 r.SHA=pinned;r.Bytes=159914
 if _,e:=anchored(b,r);e==nil{t.Fatal("source hash assertion substitute accepted")}
}
func TestSourceHostiles(t *testing.T){
 b,r:=synthetic()
 r.Candidates[0].Evidence="not present"
 if _,e:=anchored(b,r);e==nil{t.Fatal("unsupported source claim accepted")}
 r.Candidates[0].Evidence="Bar 4. No. 1 advances and <script>alert('bad')</script>."
 r.Candidates[0].Measurement="VERIFIED_3D"
 if _,e:=anchored(b,r);e==nil{t.Fatal("fabricated kinematic claim accepted")}
 r.Candidates[0].Measurement="UNMEASURED_TEXT_INSTRUCTION"
 r.URL="https://example.invalid/not-the-source"
 if _,e:=anchored(b,r);e==nil{t.Fatal("wrong source link accepted")}
}
func TestHTMLAccessibleAndEscaped(t *testing.T){
 v:=Inspection{Schema:"scp.choreography-inspection-readmodel.v0",SourceID:"pg:12926",Status:"UNVALIDATED_SOURCE_TEXT_CANDIDATES",Count:1,
 Views:[]EvidenceView{{Index:0,Line:12,Evidence:"<script>alert(1)</script>",ContextStart:11,ContextEnd:13,
 Context:"prior dance paragraph\n<script>bad source</script>\nnext line",
 CandidateID:"source-test",RoleInterpretation:"ACTOR_OR_DIRECTION_AMBIGUOUS_NOT_RESOLVED",
 ContextStatus:"ADJACENT_SOURCE_LINES_NOT_VERIFIED_FIGURE"}}}
 var w bytes.Buffer
 if e:=renderHTML(&w,v);e!=nil{t.Fatal(e)}
 html:=w.String()
 for _,required:=range []string{"<main>","<h1>","<ol class=\"results\"","<details open","<summary","<blockquote class=\"source-context\"","role=\"status\"","&lt;script&gt;"}{
  if !strings.Contains(html,required){t.Fatalf("missing %q",required)}
 }
 if strings.Contains(html,"<script>"){t.Fatal("unescaped source injection")}
 var cap bytes.Buffer
 if e:=run([]string{"-capabilities"},&cap);e!=nil{t.Fatal(e)}
 var c map[string]any
 if e:=json.Unmarshal(cap.Bytes(),&c);e!=nil||c["agent"]==""{t.Fatal("agent discovery invalid")}
 if e:=run([]string{"-capabilities","-source=x"},&cap);e==nil{t.Fatal("capability input injection accepted")}
}

func TestContextWindowIncludesNearbyOriginalLines(t *testing.T){
 lines:=[]string{"first movement","second direction","third timing","","separate figure"}
 start,end,excerpt:=contextWindow(lines,2)
 if start!=1||end!=3||excerpt!="first movement\nsecond direction\nthird timing" {t.Fatalf("wrong bounded original context %d..%d %q",start,end,excerpt)}
 if roleCue([]string{"opposite"})!="ACTOR_OR_DIRECTION_AMBIGUOUS_NOT_RESOLVED"{t.Fatal("direction silently became partner")}
}

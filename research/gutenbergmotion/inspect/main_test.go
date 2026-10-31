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

func TestSharedSearchSurfaceAndRefusal(t *testing.T){
 original:=Inspection{Schema:"scp.choreography-inspection-readmodel.v0",Count:3,VisibleCount:3,
 Candidates:[]Candidate{{ID:"first"},{ID:"second"},{ID:"third"}},
 Views:[]EvidenceView{
  {Index:0,CandidateID:"first",SearchText:"cross over right shoulder"},
  {Index:1,CandidateID:"second",SearchText:"step backward toward front"},
  {Index:2,CandidateID:"third",SearchText:"right shoulder on bar 4"},
 }}
 filtered,e:=filterInspection(original," RiGhT ShOuLdEr ")
 if e!=nil{t.Fatal(e)}
 if filtered.Count!=3||filtered.VisibleCount!=2||len(filtered.Views)!=2||len(filtered.Candidates)!=2||
 filtered.Candidates[0].ID!="first"||filtered.Candidates[1].ID!="third"{t.Fatalf("source filter mismatch %+v",filtered)}
 empty,e:=filterInspection(original,"not a source phrase");if e!=nil{t.Fatal(e)}
 if empty.VisibleCount!=0||len(empty.Views)!=0||len(empty.Candidates)!=0{t.Fatal("unmatched filter fabricated results")}
 every,e:=filterInspection(original,"");if e!=nil||every.VisibleCount!=3{t.Fatal("unfiltered corpus lost")}
 if _,e:=filterInspection(original,strings.Repeat("x",81));e==nil{t.Fatal("unbounded agent query accepted")}
 if original.Count!=3||original.VisibleCount!=3||len(original.Candidates)!=3 {t.Fatal("read operation mutated shared authority")}
}

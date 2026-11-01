package main
import (
 "bytes"
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "fmt"
 "os"
 "path/filepath"
 "strings"
 "testing"
)
func fixture()([]byte,Report){
 s:=[]byte("Bar 4. No. 1 advances and turns.\nthen comes back.\n\nBar 8. The dancers cross to the opposite side.\nand continue onward.\n")
 h:=sha256.Sum256(s)
 r:=Report{Schema:"cpe.motion-text-candidates.v0",SourceID:"pg:12926",GitCommit:sourceCommit,GitBlob:sourceBlob,SourceSHA256:hex.EncodeToString(h[:]),InputBytes:len(s),Candidates:[]Candidate{
 {ID:"first",Line:1,Evidence:"Bar 4. No. 1 advances and turns.",Actors:[]string{"no. 1"},Status:"SOURCE_TEXT_RESEARCH_CANDIDATE",Measurement:"UNMEASURED_TEXT_INSTRUCTION"},
 {ID:"second",Line:4,Evidence:"Bar 8. The dancers cross to the opposite side.",Actors:[]string{"opposite"},Status:"SOURCE_TEXT_RESEARCH_CANDIDATE",Measurement:"UNMEASURED_TEXT_INSTRUCTION"}}}
 r.Candidates[0].ID=sourceCandidateID(1,"Bar 4. No. 1 advances and turns.")
 r.Candidates[1].ID=sourceCandidateID(4,"Bar 8. The dancers cross to the opposite side.")
 return s,r
}
func sourceCandidateID(line int,source string)string{
 h:=sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%s","pg:12926",line,source)))
 return "pg12926-motion-"+hex.EncodeToString(h[:8])
}
func TestContextAndUnknown(t *testing.T){
 s,r:=fixture();o,e:=Enrich(s,r,false);if e!=nil{t.Fatal(e)}
 if o.CandidateCount!=2||o.CompleteFigureClaim||o.MotionMeasurementClaim{t.Fatalf("overclaimed %+v",o)}
 if o.ReviewGroupCount!=2||len(o.ReviewGroups)!=2 {t.Fatalf("two independent paragraph windows should not coalesce %+v",o.ReviewGroups)}
 if o.ReviewGroups[0].Status!="COLOCATED_LEXICAL_WINDOWS_NOT_A_VERIFIED_FIGURE" {t.Fatal("review window promoted to choreography")}
 if o.Windows[0].StartLine!=1||o.Windows[0].EndLine!=2||o.Windows[0].RoleStatus!="NUMBERED_ROLE_UNREVIEWED"{t.Fatalf("first %+v",o.Windows[0])}
 if o.Windows[1].StartLine!=4||o.Windows[1].EndLine!=5||o.Windows[1].RoleStatus!="AMBIGUOUS_SOURCE_ROLE"{t.Fatalf("second %+v",o.Windows[1])}
 b,_:=json.Marshal(o);o2,e:=Enrich(s,r,false);if e!=nil{t.Fatal(e)};c,_:=json.Marshal(o2);if !bytes.Equal(b,c){t.Fatal("replay changed")}
}
func TestHostiles(t *testing.T){
 s,r:=fixture()
 if _,e:=Enrich(s,r,true);e==nil{t.Fatal("wrong pinned bytes accepted")}
 bad:=append([]byte(nil),s...);bad[0]='X';if _,e:=Enrich(bad,r,false);e==nil{t.Fatal("swapped input accepted")}
 mutations:=[]func(*Report){func(q *Report){q.GitCommit="fake"},func(q *Report){q.GitBlob="fake"},func(q *Report){q.SourceID="fake"},func(q *Report){q.Candidates[0].Evidence="forged"},func(q *Report){q.Candidates[0].Line=3},func(q *Report){q.Candidates[1].ID=q.Candidates[0].ID},
  func(q *Report){q.Candidates[0].ID="pg12926-motion-deadbeefdeadbeef"},func(q *Report){q.Candidates[0].Measurement="MEASURED_MOCAP"}}
 for i,m:=range mutations{q:=r;q.Candidates=append([]Candidate(nil),r.Candidates...);m(&q);if _,e:=Enrich(s,q,false);e==nil{t.Fatalf("hostile %d accepted",i)}}
}
func TestCLIRejectsUnpinnedWithoutOutput(t *testing.T){
 s,r:=fixture();dir:=t.TempDir();src:=filepath.Join(dir,"src.txt");js:=filepath.Join(dir,"candidates.json")
 if e:=os.WriteFile(src,s,0600);e!=nil{t.Fatal(e)}
 raw,_:=json.Marshal(r);if e:=os.WriteFile(js,raw,0600);e!=nil{t.Fatal(e)}
 var out bytes.Buffer
 if e:=run([]string{"-source",src,"-candidates",js},&out);e==nil{t.Fatal("wrong book accepted")}
 if out.Len()!=0{t.Fatal("partial result emitted")}
 if e:=os.WriteFile(js,append(raw,[]byte("\n{}")...),0600);e!=nil{t.Fatal(e)}
 if e:=run([]string{"-source",src,"-candidates",js},&out);e==nil||!strings.Contains(e.Error(),"trailing"){t.Fatalf("trailing JSON accepted %v",e)}
}

func TestOverlappingWindowsAreNotIndependentFigures(t *testing.T){
 wins:=[]Window{
  {CandidateID:"near",AnchorLine:10,StartLine:8,EndLine:11},
  {CandidateID:"nearer",AnchorLine:12,StartLine:10,EndLine:14},
  {CandidateID:"independent",AnchorLine:30,StartLine:28,EndLine:32},
 }
 groups,e:=overlappingWindows(wins);if e!=nil{t.Fatal(e)}
 if len(groups)!=2||len(groups[0].CandidateIDs)!=2||groups[0].EndLine!=14{t.Fatalf("failed to coalesce overlapping contexts %+v",groups)}
 if groups[0].Status!="COLOCATED_LEXICAL_WINDOWS_NOT_A_VERIFIED_FIGURE"{t.Fatal("promoted guessed figure")}
 wins[0],wins[2]=wins[2],wins[0]
 again,e:=overlappingWindows(wins);if e!=nil{t.Fatal(e)}
 a,_:=json.Marshal(groups);b,_:=json.Marshal(again)
 if !bytes.Equal(a,b){t.Fatal("candidate order altered deterministic group ID or digest")}
 if _,e:=overlappingWindows([]Window{{CandidateID:"fake",StartLine:12,EndLine:11}});e==nil{t.Fatal("reversed interval admitted")}
}

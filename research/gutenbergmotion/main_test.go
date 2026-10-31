package main

import (
 "bytes"
 "strings"
 "testing"
 "os"
 "path/filepath"
)
func TestSourceBoundMotionCandidates(t *testing.T){
 b:=[]byte("Bar 4. Nos. 1 and 6 advance and tap sticks.\nAt bar 8 all jump.\nThe king advanced his argument at bar 4.\nNo. 2 passes the book without timing.\n")
 r,err:=Mine(b,100);if err!=nil{t.Fatal(err)}
 if len(r.Candidates)!=2{t.Fatalf("want 2 source-grounded candidates, got %d: %+v",len(r.Candidates),r.Candidates)}
 if r.Candidates[0].Line!=1||r.Candidates[1].Line!=2{t.Fatalf("line origins wrong %+v",r.Candidates)}
 for _,c:=range r.Candidates{if c.Measurement!="UNMEASURED_TEXT_INSTRUCTION"||c.Status!="SOURCE_TEXT_RESEARCH_CANDIDATE"{t.Fatalf("fabricated measurement: %+v",c)}}
}
func TestUnmeasuredNegativesAndBounds(t *testing.T){
 for _,s:=range []string{"","she danced warmly","the king advanced his argument at bar 4","all advance without timing","bar 4 No. 3 holds a thought"}{
  if s==""{if _,err:=Mine([]byte(s),2);err==nil{t.Fatal("empty accepted")};continue}
  r,err:=Mine([]byte(s),2);if err!=nil{t.Fatal(err)};if len(r.Candidates)!=0{t.Fatalf("false positive: %q",s)}
 }
 if _,err:=Mine(bytes.Repeat([]byte("x"),maxSourceBytes+1),20);err==nil{t.Fatal("oversize accepted")}
 if _,err:=Mine([]byte("bar 4 No. 1 jump"),0);err==nil{t.Fatal("zero limit accepted")}
}
func TestDeterministicIDsAndCap(t *testing.T){
 src:=[]byte("Bar 4. No. 1 jumps.\nBar 5. No. 2 advances.\n")
 a,e:=Mine(src,1);if e!=nil{t.Fatal(e)};b,e:=Mine(src,1);if e!=nil{t.Fatal(e)}
 if len(a.Candidates)!=1||!a.Truncated||a.Candidates[0].ID!=b.Candidates[0].ID{t.Fatalf("not deterministic %+v %+v",a,b)}
 if !strings.Contains(strings.Join(a.Candidates[0].Motions,","),"jump"){t.Fatal("motion term missing")}
}

func TestCLIRefusesWrongSourceBytes(t *testing.T) {
 path:=filepath.Join(t.TempDir(),"swapped.txt")
 if err:=os.WriteFile(path,[]byte("Bar 4. No. 1 jumps.\n"),0600);err!=nil{t.Fatal(err)}
 var buf bytes.Buffer
 err:=run([]string{"-input",path},&buf)
 if err==nil || !strings.Contains(err.Error(),"source bytes do not match"){t.Fatalf("expected source pin rejection, got: %v",err)}
 if buf.Len()!=0{t.Fatal("forged report emitted")}
}

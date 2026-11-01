package main
import (
 "bytes"
 "encoding/json"
 "strings"
 "testing"
)
func crossOverSyntheticSource()[]byte{
 lines:=make([]string,1470)
 for i:=range lines{lines[i]=""}
 entries:=map[int]string{
  1433:"CROSS-OVER.",
  1435:"This movement is executed in Front formation, to eight bars of music, as",
  1438:"In the first two bars each dancer crosses over and takes the place of his partner,",
  1445:"In crossing and re-crossing, right shoulder of each passes by the right shoulder of each partner.",
  1449:"In bars 3 and 4 all face inward again, by turning to the rightward.",
  1457:"In bars 5 and 6 all cross over again, right shoulder to right shoulder.",
  1460:"In bars 7 and 8 all make a full turn to the right.",
  1463:"The movement completes just in time for the",
  1465:"jump in bars 4 and 8.",
  1467:"This is called in the Notation--Cross-over.",
 }
 for i,v:=range entries{lines[i-1]=v}
 return []byte(strings.Join(lines,"\n"))
}
func TestCrossOverFirstPassSourceOnly(t *testing.T){
 original:=crossOverSyntheticSource()
 r,e:=CrossOverFirstPass(original,false);if e!=nil{t.Fatal(e)}
 if r.WrittenBars!=8||len(r.Phrases)!=7||r.DiagramsStatus!="ILLUSTRATIONS_NOT_DIGITALLY_QUALIFIED"||
 r.IndependentReview!="PENDING_SECOND_READER"||r.MusicScoreStatus!="NO_NOTATED_SCORE_BYTES_ADMITTED"||r.PhysicalMotion!="UNKNOWN_NOT_MEASURED"{
  t.Fatalf("missing acceptance hold: %+v",r)
 }
 a,_:=json.Marshal(r);again,e:=CrossOverFirstPass(original,false);if e!=nil{t.Fatal(e)};b,_:=json.Marshal(again)
 if !bytes.Equal(a,b){t.Fatal("unrepeatable source figure receipt")}
 if _,e:=CrossOverFirstPass(original,true);e==nil{t.Fatal("forged edition accepted")}
 wrong:=strings.Replace(string(original),"all cross over again","all wave their arms",1)
 if _,e:=CrossOverFirstPass([]byte(wrong),false);e==nil{t.Fatal("forged bar5-6 event accepted")}
 wrong=strings.Replace(string(original),"CROSS-OVER.","BACK-TO-BACK.",1)
 if _,e:=CrossOverFirstPass([]byte(wrong),false);e==nil{t.Fatal("forged heading accepted")}
}
func TestCrossOverCLIRejectsFabricatedSource(t *testing.T){
 var b bytes.Buffer
 if e:=run([]string{"-cross-over-first-pass","-source","/nonexistent"},&b);e==nil{t.Fatal("missing source accepted")}
 if b.Len()!=0{t.Fatal("partial first-pass output")}
 if e:=run([]string{"-cross-over-first-pass","-candidates","fake","-source","fake"},&b);e==nil{t.Fatal("mixed source & report accepted")}
}

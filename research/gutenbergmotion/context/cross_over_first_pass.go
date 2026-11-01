package main

// A first-pass textual reference to a COMPLETE eight-bar section of the
// pinned public historical work. The omitted diagrams and independent review
// are hard holds; no pose/foot contact is inferred.
import (
 "crypto/sha256"
 "encoding/hex"
 "errors"
 "fmt"
 "strings"
)
type PhraseWitness struct{
 ID string `json:"id"`
 Bars string `json:"bars"`
 StartLine int `json:"start_line"`
 EndLine int `json:"end_line"`
 Anchor string `json:"anchor"`
 Interpretation string `json:"interpretation_status"`
}
type FigureWitness struct{
 Schema string `json:"schema"`
 SourceSHA256 string `json:"source_sha256"`
 Name string `json:"name"`
 FigureStartLine int `json:"figure_start_line"`
 FigureEndLine int `json:"figure_end_line"`
 WrittenBars int `json:"written_bars"`
 SourceEdition string `json:"source_edition"`
 TextStatus string `json:"text_status"`
 DiagramsStatus string `json:"diagrams_status"`
 MusicScoreStatus string `json:"music_score_status"`
 IndependentReview string `json:"independent_review"`
 PhysicalMotion string `json:"physical_motion"`
 Phrases []PhraseWitness `json:"phrases"`
}
func CrossOverFirstPass(src []byte, requirePin bool)(FigureWitness,error){
 if len(src)==0||len(src)>maxSource{return FigureWitness{},errors.New("source size rejected")}
 h:=sha256.Sum256(src);digest:=hex.EncodeToString(h[:])
 if requirePin&&digest!=sourceSHA{return FigureWitness{},errors.New("not the original pinned edition")}
 lines,err:=linesOf(src);if err!=nil{return FigureWitness{},err}
 phrases:=[]PhraseWitness{
  {ID:"figure-eight-bars",Bars:"1-8",StartLine:1433,EndLine:1436,Anchor:"This movement is executed in Front formation, to eight bars of music",Interpretation:"SOURCE_TEXT_ONLY"},
  {ID:"partner-cross",Bars:"1-2",StartLine:1438,EndLine:1439,Anchor:"each dancer crosses over and takes the place of his partner",Interpretation:"DIRECTION_AND_PARTNER_NOT_A_JOINT_PATH"},
  {ID:"right-shoulder-rule",Bars:"1-8",StartLine:1445,EndLine:1447,Anchor:"right shoulder of each passes by the right shoulder",Interpretation:"SOURCE_TEXT_RULE_NO_CONTACT_CLAIM"},
  {ID:"right-turn-inward",Bars:"3-4",StartLine:1449,EndLine:1451,Anchor:"face inward again, by turning to the rightward",Interpretation:"SOURCE_TEXT_ONLY"},
  {ID:"recross",Bars:"5-6",StartLine:1457,EndLine:1458,Anchor:"all cross over again, right shoulder to right shoulder",Interpretation:"SOURCE_TEXT_ONLY"},
  {ID:"right-turn-front",Bars:"7-8",StartLine:1460,EndLine:1461,Anchor:"all make a full turn to the right",Interpretation:"SOURCE_TEXT_ONLY"},
  {ID:"half-bar-jump-cue",Bars:"4,8",StartLine:1463,EndLine:1465,Anchor:"jump in bars 4 and 8",Interpretation:"HALF_BAR_CUE_NO_KINEMATIC_CLAIM"},
 }
 if len(lines)<1467||strings.TrimSpace(lines[1433-1])!="CROSS-OVER."||
 strings.TrimSpace(lines[1467-1])!="This is called in the Notation--Cross-over."{
  return FigureWitness{},errors.New("figure heading or ending source anchor mismatch")
 }
 for _,p:=range phrases{
  if p.StartLine<1433||p.EndLine>1467||p.EndLine<p.StartLine{return FigureWitness{},errors.New("invalid figure span")}
  actual:=strings.Join(lines[p.StartLine-1:p.EndLine]," ")
  if !strings.Contains(actual,p.Anchor){return FigureWitness{},fmt.Errorf("missing witness for %s at %d-%d",p.ID,p.StartLine,p.EndLine)}
 }
 return FigureWitness{Schema:"scp.source-figure-first-pass.v0",SourceSHA256:digest,Name:"Cross-over",
 FigureStartLine:1433,FigureEndLine:1467,WrittenBars:8,
 SourceEdition:"pg:12926@1d5ca33372491509805a64cd0425a5164d3108fa",
 TextStatus:"SOURCE_SPANS_CHECKED_FIRST_PASS_NOT_INDEPENDENT_GOLD",
 DiagramsStatus:"ILLUSTRATIONS_NOT_DIGITALLY_QUALIFIED",
 MusicScoreStatus:"NO_NOTATED_SCORE_BYTES_ADMITTED",
 IndependentReview:"PENDING_SECOND_READER",PhysicalMotion:"UNKNOWN_NOT_MEASURED",
 Phrases:phrases},nil
}

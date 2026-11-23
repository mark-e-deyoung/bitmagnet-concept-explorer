// cpeproposal is an OFFLINE, research-only independent Go validator of the
// Python R36 ConceptDiscoveryProposal envelope. It is NOT Concept Authority,
// a promotion/intake server, a rights issuer, an adjudicator or a reconciler.
// Inputs must be ORIGINAL PUBLIC SYNTHETIC in this experiment.
package main

import (
    "bytes"
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
    "unicode/utf8"
)

const (
    proposalVersion = "cpe-discovery-proposal-research-v0"
    maxProposalBytes = 65536
    maxJSONDepth = 128
)
var (
    sha64 = regexp.MustCompile("^[0-9a-f]{64}$")
    sha40 = regexp.MustCompile("^[0-9a-f]{40}$")
    typedID = regexp.MustCompile("^[A-Za-z0-9][A-Za-z0-9:._-]{0,159}$")
    locale = regexp.MustCompile("^[a-z]{2,3}(-[A-Za-z0-9]{2,8}){0,3}$")
)

type Expected struct {
    SourceLockSHA256 string
    SnapshotBytesSHA256 string
    ProducerRevision string
    EvidenceSHA256 string
    SourceSystem string
}
type Report struct {
    Status string `json:"status"`
    Authority string `json:"authority"`
    ProposalID string `json:"proposal_id"`
    Kind string `json:"proposal_kind"`
    Disposition string `json:"disposition"`
    Equivalence string `json:"equivalence_status"`
    IndependentCorroboration string `json:"independent_corroboration"`
    AcceptedSemanticChanges int `json:"accepted_semantic_changes"`
}

func fail(reason string) error { return errors.New(reason) }

func jsonWalk(dec *json.Decoder, depth int) error {
    if depth > maxJSONDepth { return fail("JSON_DEPTH_LIMIT") }
    tok, err := dec.Token()
    if err != nil { return fail("MALFORMED_JSON_TOKEN") }
    d, ok := tok.(json.Delim)
    if !ok { return nil }
    switch d {
    case '{':
        seen := map[string]bool{}
        for dec.More() {
            key, err := dec.Token()
            if err != nil { return fail("MALFORMED_JSON_KEY") }
            name, ok := key.(string)
            if !ok { return fail("MALFORMED_JSON_KEY") }
            if seen[name] { return fail("DUPLICATE_JSON_KEY") }
            seen[name] = true
            if err := jsonWalk(dec,depth+1); err != nil { return err }
        }
        end,err:=dec.Token()
        if err!=nil || end!=json.Delim('}') { return fail("MALFORMED_JSON_OBJECT") }
    case '[':
        for dec.More() {
            if err := jsonWalk(dec,depth+1); err != nil { return err }
        }
        end,err:=dec.Token()
        if err!=nil || end!=json.Delim(']') { return fail("MALFORMED_JSON_ARRAY") }
    default:
        return fail("MALFORMED_JSON_DELIMITER")
    }
    return nil
}
func strictDecode(raw []byte) (map[string]any,error) {
    if len(raw)>maxProposalBytes { return nil,fail("OVERSIZED_PROPOSAL") }
    if !utf8.Valid(raw) { return nil,fail("INVALID_UTF8") }
    tokens:=json.NewDecoder(bytes.NewReader(raw))
    if err:=jsonWalk(tokens,0);err!=nil { return nil,err }
    if _,err:=tokens.Token();err!=io.EOF {
        return nil,fail("TRAILING_JSON")
    }
    d:=json.NewDecoder(bytes.NewReader(raw))
    d.UseNumber()
    var m map[string]any
    if err:=d.Decode(&m);err!=nil || m==nil {
        return nil,fail("MALFORMED_PROPOSAL")
    }
    if err:=d.Decode(new(any));err!=io.EOF { return nil,fail("TRAILING_JSON") }
    return m,nil
}
func exactObject(v any, keys ...string) (map[string]any,error) {
    obj,ok:=v.(map[string]any)
    if !ok || len(obj)!=len(keys) {return nil,fail("UNSUPPORTED_PROPOSAL_SHAPE")}
    for _,k:=range keys { if _,ok:=obj[k];!ok {return nil,fail("UNSUPPORTED_PROPOSAL_SHAPE")} }
    return obj,nil
}
func str(v any) string { s,_:=v.(string);return s }
func validHash(s string) bool {return sha64.MatchString(s)}
func expectedValid(e Expected) bool {
    return validHash(e.SourceLockSHA256) && validHash(e.SnapshotBytesSHA256) &&
        sha40.MatchString(e.ProducerRevision) && validHash(e.EvidenceSHA256) && typedID.MatchString(e.SourceSystem)
}
func eqBool(obj map[string]any,k string,b bool) bool {v,ok:=obj[k].(bool);return ok && v==b}

func validate(raw []byte, expected Expected) (Report,error) {
    empty:=Report{}
    if !expectedValid(expected) {return empty,fail("INVALID_EXPECTED_PINS")}
    doc,err:=strictDecode(raw)
    if err!=nil {return empty,err}
    root,err:=exactObject(doc,"proposal_id","schema_version","authority",
        "adjudication","proposal_kind","producer","source_projection",
        "observation","evidence","mapping","privacy")
    if err!=nil {return empty,err}
    if str(root["schema_version"])!=proposalVersion ||
        str(root["authority"])!="RESEARCH_PROPOSAL_ONLY" ||
        str(root["adjudication"])!="UNADJUDICATED" {
        return empty,fail("UNAUTHORIZED_SEMANTIC_AUTHORITY")
    }
    producer,err:=exactObject(root["producer"],"component","exact_revision")
    if err!=nil{return empty,err}
    if str(producer["component"])!="bce.independent_python_consumer.r35" ||
        str(producer["exact_revision"])!=expected.ProducerRevision {
        return empty,fail("PRODUCER_IDENTITY_MISMATCH")
    }
    source,err:=exactObject(root["source_projection"],"authority","source_lock_sha256","snapshot_bytes_sha256")
    if err!=nil {return empty,err}
    if str(source["authority"])!="research_only" ||
        str(source["source_lock_sha256"])!=expected.SourceLockSHA256 ||
        str(source["snapshot_bytes_sha256"])!=expected.SnapshotBytesSHA256 {
        return empty,fail("SOURCE_PROJECTION_PIN_MISMATCH")
    }
    evidence,err:=exactObject(root["evidence"],"provenance_digest",
        "source_ancestry","support_count","independent_corroboration",
        "confidence","falsifier_status")
    if err!=nil{return empty,err}
    ancestry,ok:=evidence["source_ancestry"].([]any)
    if !ok || len(ancestry)!=1 || str(ancestry[0])!=expected.EvidenceSHA256 ||
        str(evidence["provenance_digest"])!=expected.EvidenceSHA256 ||
        str(evidence["independent_corroboration"])!="NOT_ESTABLISHED" ||
        evidence["support_count"]!=json.Number("1") ||
        evidence["confidence"]!=nil ||
        str(evidence["falsifier_status"])!="NOT_YET_CHALLENGED" {
        return empty,fail("UNQUALIFIED_EVIDENCE_ANCESTRY")
    }
    privacy,err:=exactObject(root["privacy"],"class",
        "raw_private_content_included","sanitization_independently_qualified")
    if err!=nil{return empty,err}
    if str(privacy["class"])!="synthetic_public_only" ||
        !eqBool(privacy,"raw_private_content_included",false) ||
        !eqBool(privacy,"sanitization_independently_qualified",false) {
        return empty,fail("UNQUALIFIED_PRIVACY_CLAIM")
    }
    mapping,err:=exactObject(root["mapping"],"candidate_relation",
        "equivalence_status","classification","candidate_not_accepted")
    if err!=nil{return empty,err}
    if str(mapping["candidate_relation"])!="unmapped" ||
        str(mapping["equivalence_status"])!="UNKNOWN" ||
        !eqBool(mapping,"candidate_not_accepted",true) {
        return empty,fail("UNAUTHORIZED_MAPPING_PROMOTION")
    }
    obs,err:=exactObject(root["observation"],"target_concept_id","language",
        "observation_type","native_term","consumer_status","consumer_impact","source_system")
    if err!=nil{return empty,err}
    if !typedID.MatchString(str(obs["target_concept_id"])) ||
        !locale.MatchString(str(obs["language"])) ||
        str(obs["source_system"])!=expected.SourceSystem ||
        str(obs["consumer_impact"])!="RETRIEVAL_MISS_OR_MAPPING_GAP" {
        return empty,fail("INVALID_OBSERVATION")
    }
    kind:=str(root["proposal_kind"])
    classification:=str(mapping["classification"])
    status:=str(obs["consumer_status"])
    typ:=str(obs["observation_type"])
    term:=obs["native_term"]
    if kind=="lexicalization_candidate" {
        value,ok:=term.(string)
        if !ok || value=="" || len([]rune(value))>96 ||
            strings.ContainsAny(value,"/\\") || strings.Contains(value,"://") ||
            strings.ContainsRune(value,'\uFFFD') ||
            typ!="native_source_term" ||
            (status!="NO_LEXICALIZATION" && status!="OBSERVED") ||
            classification!="NATIVE_SOURCE_TERM_NOT_IN_PINNED_LEXICAL_VIEW" {
            return empty,fail("INVALID_LEXICAL_CANDIDATE")
        }
    } else if kind=="coverage_gap" {
        if classification=="UNKNOWN_CONCEPT_NOT_PROOF_OF_NOVELTY" {
            if status!="UNKNOWN_CONCEPT" {return empty,fail("INVALID_GAP_CLASSIFICATION")}
        } else if classification=="KNOWN_CONCEPT_RETRIEVAL_MISS_NOT_ONTOLOGY_GAP" {
            if typ!="consumer_lookup_miss" || term!=nil || status=="UNKNOWN_CONCEPT" {
                return empty,fail("INVALID_GAP_CLASSIFICATION")
            }
        } else {return empty,fail("INVALID_GAP_CLASSIFICATION")}
    } else {
        return empty,fail("UNSUPPORTED_PROPOSAL_KIND")
    }
    pid:=str(root["proposal_id"])
    if !strings.HasPrefix(pid,"proposal:sha256:") || !validHash(strings.TrimPrefix(pid,"proposal:sha256:")) {
        return empty,fail("INVALID_PROPOSAL_ID")
    }
    delete(root,"proposal_id")
    buf:=new(bytes.Buffer)
    encoder:=json.NewEncoder(buf)
    encoder.SetEscapeHTML(false)
    if err:=encoder.Encode(root);err!=nil {return empty,fail("INVALID_CANONICAL_BODY")}
    canonical:=bytes.TrimSuffix(buf.Bytes(),[]byte("\n"))
    digest:=sha256.Sum256(append([]byte(proposalVersion+"\x00"),canonical...))
    expectedID:="proposal:sha256:"+hex.EncodeToString(digest[:])
    if pid!=expectedID {
        return empty,fail("PROPOSAL_IDENTITY_MISMATCH")
    }
    return Report{Status:"VALID_RESEARCH_PROPOSAL_ONLY",
        Authority:"RESEARCH_ONLY", ProposalID:pid, Kind:kind,
        Disposition:"DEFER_INDEPENDENT_GROUNDING",
        Equivalence:"UNKNOWN", IndependentCorroboration:"NOT_ESTABLISHED",
        AcceptedSemanticChanges:0},nil
}
func main() {
    file:=flag.String("file","","research proposal JSON file")
    source:=flag.String("expected-source-lock-sha256","","independent source lock pin")
    snap:=flag.String("expected-snapshot-sha256","","independent exact snapshot byte pin")
    producer:=flag.String("expected-producer-revision","","independent expected producer Git revision")
    evidence:=flag.String("expected-evidence-sha256","","independent source witness SHA256")
    system:=flag.String("expected-source-system","","expected source-system ID")
    flag.Parse()
    if *file=="" || flag.NArg()!=0 {
        fmt.Fprintln(os.Stderr,"INVALID_ARGUMENTS");os.Exit(2)
    }
    f,err:=os.Open(*file)
    if err!=nil {fmt.Fprintln(os.Stderr,"UNREADABLE_PROPOSAL");os.Exit(3)}
    defer f.Close()
    raw,err:=io.ReadAll(io.LimitReader(f,maxProposalBytes+1))
    if err!=nil {fmt.Fprintln(os.Stderr,"UNREADABLE_PROPOSAL");os.Exit(3)}
    report,err:=validate(raw,Expected{*source,*snap,*producer,*evidence,*system})
    if err!=nil {fmt.Fprintln(os.Stderr,err.Error());os.Exit(3)}
    enc:=json.NewEncoder(os.Stdout)
    enc.SetEscapeHTML(false)
    if err:=enc.Encode(report);err!=nil {fmt.Fprintln(os.Stderr,"OUTPUT_FAILURE");os.Exit(3)}
}

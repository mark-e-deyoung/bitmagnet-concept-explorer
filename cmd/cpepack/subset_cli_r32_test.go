package main

import (
    "bytes"
    "encoding/json"
    "os"
    "path/filepath"
    "strings"
    "testing"

    "github.com/mark-e-deyoung/bitmagnet-concept-explorer/internal/cpepack"
)

// All data below is the repository's original public-safe synthetic fixture.
// These tests prove API/CLI consistency, NOT an independently admitted release.
type r32Fixture struct {
    packages []cpepack.Package
    lock cpepack.Lock
    selector cpepack.SubsetSelector
    refs []cpepack.RepresentationRef
    root string
    packagePath string
    lockPath string
    selectorPath string
    refsPath string
}
// JSON is the exported consumer contract: the Go library may use nil or
// non-nil empty slices internally while the wire representation is identical.
// Do not use reflect.DeepEqual across a JSON round-trip (original R32 false RED).
func r32JSONEqual(t *testing.T, a, b any) bool {
    t.Helper()
    left, err := json.Marshal(a)
    if err != nil {t.Fatal(err)}
    right, err := json.Marshal(b)
    if err != nil {t.Fatal(err)}
    return bytes.Equal(left, right)
}
func r32LoadFixture(t *testing.T) r32Fixture {
    t.Helper()
    dir := filepath.Join("..", "..", "testdata", "cpepack")
    f := r32Fixture{
        packagePath: filepath.Join(dir, "packages.json"),
        lockPath: filepath.Join(dir, "lock.json"),
        root: "test.public.literature",
        selector: cpepack.SubsetSelector{
            SchemaVersion: cpepack.SubsetVersion,
            ProfileID: "fixture.cli.r32",
            RootConceptIDs: []string{"public:literary-work"},
            RequiredEdges: []cpepack.RequiredEdge{{FromID:"public:literary-work", Predicate:"subclass_of", TargetID:"public:work"}},
        },
        refs: []cpepack.RepresentationRef{{
            SchemaVersion: cpepack.RepresentationVersion,
            ReferenceID: "fixture:literature-ref",
            ConceptID: "public:literary-work",
            Role: "demonstrates",
            Modality: "text",
            MediaType: "application/json",
            FormatVersion: "fixture.text.v0",
            Artifact: cpepack.ArtifactIdentity{Kind:"raw-sha256", Digest: "sha256:" + strings.Repeat("a", 64)},
            Visibility: "public",
            Rights: "redistributable",
            SourceRevision: "fixture-source-v1",
        }},
    }
    if err := read(f.packagePath, &f.packages); err != nil { t.Fatal(err) }
    if err := read(f.lockPath, &f.lock); err != nil { t.Fatal(err) }
    tmp := t.TempDir()
    write := func(name string, data any) string {
        p := filepath.Join(tmp,name)
        raw, err := json.Marshal(data)
        if err != nil { t.Fatal(err) }
        if err := os.WriteFile(p,raw,0600); err != nil { t.Fatal(err) }
        return p
    }
    f.selectorPath = write("selector.json",f.selector)
    f.refsPath = write("refs.json",f.refs)
    return f
}
func (f r32Fixture) args() []string {
    return []string{"-lock",f.lockPath,"-packages",f.packagePath,"-root",f.root}
}
func (f r32Fixture) subsetArgs() []string {
    return append(f.args(), "-selector",f.selectorPath,"-representations",f.refsPath)
}
func r32Run(t *testing.T, args []string) (int, string, string) {
    t.Helper()
    var stdout,stderr bytes.Buffer
    status := run(args,&stdout,&stderr)
    return status,stdout.String(),stderr.String()
}

func TestR32LegacyCLIMatchesExistingDirectResearchProjection(t *testing.T) {
    f := r32LoadFixture(t)
    direct,err := cpepack.Build(f.packages,f.lock,[]string{f.root},true)
    if err!=nil {t.Fatal(err)}
    status,out,diagnostic := r32Run(t,f.args())
    if status!=0 {t.Fatalf("legacy CLI status %d: %s",status,diagnostic)}
    var got cpepack.Projection
    if err:=json.Unmarshal([]byte(out),&got);err!=nil {t.Fatal(err)}
    if !r32JSONEqual(t,direct,got) || got.Authority!="research_only" {
        t.Fatalf("legacy CLI changed existing projection: got=%+v want=%+v",got,direct)
    }
}

func TestR32SubsetCLIMatchesExistingDirectReadLibrary(t *testing.T) {
    f := r32LoadFixture(t)
    direct,err:=cpepack.BuildSubset(f.packages,f.lock,[]string{f.root},f.selector,f.refs,true)
    if err!=nil {t.Fatal(err)}
    status,out,diagnostic:=r32Run(t,f.subsetArgs())
    if status!=0 {t.Fatalf("subset CLI status %d: %s",status,diagnostic)}
    var got cpepack.SubsetView
    if err:=json.Unmarshal([]byte(out),&got);err!=nil {t.Fatal(err)}
    if !r32JSONEqual(t,direct,got) || got.Authority!="research_only" {
        t.Fatalf("CLI and library have different content: got=%+v want=%+v",got,direct)
    }
    if len(got.Concepts)!=2 || len(got.Representations)!=1 {
        t.Fatalf("expected required-edge concept closure and one synthetic representation: %+v",got)
    }
}

func TestR32PinnedCLIAdmitsOnlyExpectedResearchSelection(t *testing.T) {
    f:=r32LoadFixture(t)
    direct,err:=cpepack.BuildSubset(f.packages,f.lock,[]string{f.root},f.selector,f.refs,true)
    if err!=nil {t.Fatal(err)}
    args:=append(f.subsetArgs(),"-expected-source-sha256",direct.SourceLockSHA256,"-expected-selection-sha256",direct.Digest)
    status,out,diagnostic:=r32Run(t,args)
    if status!=0 {t.Fatalf("pinned CLI status %d: %s",status,diagnostic)}
    var got cpepack.SubsetView
    if err:=json.Unmarshal([]byte(out),&got);err!=nil {t.Fatal(err)}
    if !r32JSONEqual(t,direct,got){t.Fatal("pinned CLI diverged from direct verified library result")}
    if got.Authority!="research_only" {t.Fatalf("unexpected authority upgrade: %s",got.Authority)}
}

func TestR32PinnedCLIRefusesWrongIndependentPin(t *testing.T) {
    f:=r32LoadFixture(t)
    direct,err:=cpepack.BuildSubset(f.packages,f.lock,[]string{f.root},f.selector,f.refs,true)
    if err!=nil {t.Fatal(err)}
    wrong:=strings.Repeat("b",64)
    for _,tc:=range []struct{name,source,selection,reason string}{
        {"source",wrong,direct.Digest,"source_pin_mismatch"},
        {"selection",direct.SourceLockSHA256,wrong,"selection_pin_mismatch"},
    }{
        t.Run(tc.name,func(t *testing.T){
            args:=append(f.subsetArgs(),"-expected-source-sha256",tc.source,"-expected-selection-sha256",tc.selection)
            status,out,diagnostic:=r32Run(t,args)
            if status!=1 || out!="" || !strings.Contains(diagnostic,tc.reason) {
                t.Fatalf("wrong pin accepted: status=%d output=%q err=%q",status,out,diagnostic)
            }
        })
    }
}

func TestR32CLIRefusesIncompletePinAndSelectorInputs(t *testing.T) {
    f:=r32LoadFixture(t)
    cases:=[][]string{
        append(f.args(),"-selector",f.selectorPath),
        append(f.args(),"-representations",f.refsPath),
        append(f.args(),"-expected-source-sha256",strings.Repeat("a",64)),
        append(f.subsetArgs(),"-expected-source-sha256",strings.Repeat("a",64)),
        append(f.subsetArgs(),"-expected-selection-sha256",strings.Repeat("a",64)),
    }
    for i,args:=range cases {
        status,out,_:=r32Run(t,args)
        if status!=2 || out!="" {t.Fatalf("incomplete args case %d accepted: %d %q",i,status,out)}
    }
}
func TestR32PublicConsumerRejectsPrivateRepresentation(t *testing.T) {
    f:=r32LoadFixture(t)
    f.refs[0].Visibility="private"
    raw,err:=json.Marshal(f.refs)
    if err!=nil {t.Fatal(err)}
    if err:=os.WriteFile(f.refsPath,raw,0600);err!=nil {t.Fatal(err)}
    status,out,diagnostic:=r32Run(t,f.subsetArgs())
    if status!=1 || out!="" || !strings.Contains(diagnostic,"private_representation_in_public_closure") {
        t.Fatalf("private reference leaked: status=%d out=%q diagnostic=%q",status,out,diagnostic)
    }
}

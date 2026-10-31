package cpepack

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOCILayoutDeterministicPublicDataOnly(t *testing.T) {
	p, l := original()
	d := t.TempDir()
	a := filepath.Join(d, "a")
	b := filepath.Join(d, "b")
	h, e := WriteOCIArtifact(a, p, l, []string{"test.public.literature"})
	if e != nil { t.Fatal(e) }
	h2, e := WriteOCIArtifact(b, []Package{p[1],p[2],p[0]}, Lock{SchemaVersion:l.SchemaVersion,Packages:[]LockedPackage{l.Packages[2],l.Packages[0],l.Packages[1]}}, []string{"test.public.literature"})
	if e != nil { t.Fatal(e) }
	if h!=h2 {t.Fatal("OCI manifest changed with package order")}
	for _,name:=range []string{"index.json","oci-layout"} {
		x,e:=os.ReadFile(filepath.Join(a,name));if e!=nil {t.Fatal(e)}
		y,e:=os.ReadFile(filepath.Join(b,name));if e!=nil||!bytes.Equal(x,y) {t.Fatalf("OCI layout mismatch %s: %v",name,e)}
	}
	raw,e:=os.ReadFile(filepath.Join(a,"blobs","sha256",strings.TrimPrefix(h,"sha256:")));if e!=nil {t.Fatal(e)}
	if "sha256:"+sha(raw)!=h {t.Fatal("manifest hash mismatch")}
	var m ociManifest
	if e=json.Unmarshal(raw,&m);e!=nil {t.Fatal(e)}
	if m.ArtifactType!=ociCPEType||m.Config.MediaType!=ociEmptyType||len(m.Layers)!=1 {t.Fatal("not a non-executable OCI data artifact")}
	blob,e:=os.ReadFile(filepath.Join(a,"blobs","sha256",strings.TrimPrefix(m.Layers[0].Digest,"sha256:")));if e!=nil {t.Fatal(e)}
	if "sha256:"+sha(blob)!=m.Layers[0].Digest||len(blob)!=m.Layers[0].Size||bytes.Contains(blob,[]byte("test.private")) {t.Fatal("data layer digest/visibility violation")}
	if _,e:=WriteOCIArtifact(a,p,l,[]string{"test.public.literature"});e==nil {t.Fatal("existing OCI output replaced")}
}
func TestOCILayoutPrivateClosureRefusedBeforeWrite(t *testing.T) {
	p,l:=original();out:=filepath.Join(t.TempDir(),"blocked")
	if _,e:=WriteOCIArtifact(out,p,l,[]string{"test.private.domain"});e==nil {t.Fatal("private data materialized")}
	if _,e:=os.Stat(out);!os.IsNotExist(e){t.Fatalf("unsafe output directory created: %v",e)}
}

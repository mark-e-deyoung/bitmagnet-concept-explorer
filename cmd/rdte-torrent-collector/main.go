package main

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Preserve real publisher metainfo. No BitTorrent connections or payload I/O.
type parser struct {
	b        []byte
	i, nodes int
}

func (p *parser) value(depth int) (any, error) {
	p.nodes++
	if depth > 64 || p.nodes > 150000 || p.i >= len(p.b) {
		return nil, errors.New("malformed bencode depth/nodes")
	}
	switch p.b[p.i] {
	case 'i':
		p.i++
		start := p.i
		for p.i < len(p.b) && p.b[p.i] != 'e' {
			p.i++
		}
		if p.i >= len(p.b) {
			return nil, errors.New("bad integer")
		}
		n, e := strconv.ParseInt(string(p.b[start:p.i]), 10, 64)
		p.i++
		return n, e
	case 'l':
		p.i++
		a := []any{}
		for p.i < len(p.b) && p.b[p.i] != 'e' {
			v, e := p.value(depth + 1)
			if e != nil {
				return nil, e
			}
			a = append(a, v)
		}
		if p.i >= len(p.b) {
			return nil, errors.New("bad list")
		}
		p.i++
		return a, nil
	case 'd':
		p.i++
		m := map[string]any{}
		for p.i < len(p.b) && p.b[p.i] != 'e' {
			v, e := p.value(depth + 1)
			if e != nil {
				return nil, e
			}
			key, ok := v.([]byte)
			if !ok {
				return nil, errors.New("bad dict key")
			}
			k := string(key)
			if _, ok = m[k]; ok {
				return nil, errors.New("duplicate key")
			}
			v, e = p.value(depth + 1)
			if e != nil {
				return nil, e
			}
			m[k] = v
		}
		if p.i >= len(p.b) {
			return nil, errors.New("bad dict")
		}
		p.i++
		return m, nil
	default:
		if p.b[p.i] < '0' || p.b[p.i] > '9' {
			return nil, errors.New("bad bencode marker")
		}
		start := p.i
		for p.i < len(p.b) && p.b[p.i] >= '0' && p.b[p.i] <= '9' {
			p.i++
		}
		if p.i >= len(p.b) || p.b[p.i] != ':' {
			return nil, errors.New("bad length")
		}
		n, e := strconv.Atoi(string(p.b[start:p.i]))
		p.i++
		if e != nil || n < 0 || n > len(p.b)-p.i {
			return nil, errors.New("invalid byte length")
		}
		v := p.b[p.i : p.i+n]
		p.i += n
		return v, nil
	}
}
func decodeMetainfo(b []byte) (hash, name string, size int64, err error) {
	if len(b) < 20 || len(b) > 2<<20 || b[0] != 'd' {
		return "", "", 0, errors.New("metainfo size/root invalid")
	}
	p := &parser{b: b, i: 1}
	var info any
	start, end := 0, 0
	seen := map[string]bool{}
	for p.i < len(b) && b[p.i] != 'e' {
		k, e := p.value(1)
		if e != nil {
			return "", "", 0, e
		}
		kb, ok := k.([]byte)
		if !ok {
			return "", "", 0, errors.New("root key not string")
		}
		key := string(kb)
		if seen[key] {
			return "", "", 0, errors.New("duplicate root key")
		}
		seen[key] = true
		begin := p.i
		v, e := p.value(1)
		if e != nil {
			return "", "", 0, e
		}
		if key == "info" {
			info = v
			start = begin
			end = p.i
		}
	}
	if p.i != len(b)-1 || b[p.i] != 'e' {
		return "", "", 0, errors.New("root not terminated")
	}
	m, ok := info.(map[string]any)
	if !ok {
		return "", "", 0, errors.New("info missing")
	}
	if _, v2 := m["meta version"]; v2 {
		return "", "", 0, errors.New("unsupported v2/hybrid")
	}
	if priv, ok := m["private"].(int64); ok && priv != 0 {
		return "", "", 0, errors.New("private metainfo excluded")
	}
	n, ok := m["name"].([]byte)
	if !ok || len(n) == 0 || len(n) > 500 {
		return "", "", 0, errors.New("name missing/bad")
	}
	if _, ok = m["pieces"].([]byte); !ok {
		return "", "", 0, errors.New("v1 pieces missing")
	}
	if length, ok := m["length"].(int64); ok {
		size = length
	} else {
		files, ok := m["files"].([]any)
		if !ok || len(files) == 0 || len(files) > 15000 {
			return "", "", 0, errors.New("file list invalid")
		}
		for _, item := range files {
			f, ok := item.(map[string]any)
			if !ok {
				return "", "", 0, errors.New("file invalid")
			}
			length, ok := f["length"].(int64)
			if !ok || length < 0 || size > 1<<62-length {
				return "", "", 0, errors.New("length invalid")
			}
			size += length
		}
	}
	if size <= 0 {
		return "", "", 0, errors.New("empty torrent")
	}
	sum := sha1.Sum(b[start:end])
	return hex.EncodeToString(sum[:]), string(n), size, nil
}
func fetch(rawurl string, max int) ([]byte, error) {
	u, e := url.Parse(rawurl)
	if e != nil || u.Scheme != "https" {
		return nil, errors.New("HTTPS mandatory")
	}
	if u.Hostname() != "archive.org" && u.Hostname() != "releases.ubuntu.com" && u.Hostname() != "mirrors.edge.kernel.org" {
		return nil, errors.New("unapproved source")
	}
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 3 {
			return errors.New("redirect budget")
		}
		host := req.URL.Hostname()
		if host == "archive.org" || host == "releases.ubuntu.com" || host == "mirrors.edge.kernel.org" || strings.HasSuffix(host, ".archive.org") {
			return nil
		}
		return errors.New("off-origin redirect")
	}}
	req, e := http.NewRequest("GET", rawurl, nil)
	if e != nil {
		return nil, e
	}
	req.Header.Set("User-Agent", "CPE-public-RDTE/0.1 metainfo-only")
	resp, e := client.Do(req)
	if e != nil {
		return nil, e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(resp.Body, int64(max+1)))
	if len(b) > max {
		return nil, errors.New("byte budget")
	}
	return b, e
}
func admit(out string, entries *[]map[string]any, source, record, rights, link, contentType string) error {
	raw, e := fetch(link, 2<<20)
	if e != nil {
		return e
	}
	hash, name, size, e := decodeMetainfo(raw)
	if e != nil {
		return e
	}
	sum := sha256.Sum256(raw)
	sha := hex.EncodeToString(sum[:])
	for _, prior := range *entries {
		if prior["metainfo_sha256"] == sha {
			return nil
		}
	}
	file := filepath.Join("metainfo", sha+".torrent")
	if e = os.WriteFile(filepath.Join(out, file), raw, 0644); e != nil {
		return e
	}
	*entries = append(*entries, map[string]any{"source": source, "source_record": record, "source_url": link, "rights_source_claim": rights, "rights_status": "source_claim_not_independently_qualified", "infohash": hash, "metainfo_sha256": sha, "metainfo_bytes": len(raw), "metainfo_file": file, "name": name, "size_bytes": size, "content_type": contentType, "p2p_used": false, "payload_downloaded": false})
	return nil
}
func collectArchive(out string, entries *[]map[string]any, events *[]string) {
	q := url.Values{}
	q.Set("q", "collection:prelinger AND mediatype:movies AND format:\"Archive BitTorrent\"")
	q.Set("output", "json")
	q.Set("rows", "20")
	q.Add("fl[]", "identifier")
	q.Add("sort[]", "downloads desc")
	data, e := fetch("https://archive.org/advancedsearch.php?"+q.Encode(), 2<<20)
	if e != nil {
		*events = append(*events, "archive discover: "+e.Error())
		return
	}
	var res struct {
		Response struct{ Docs []struct{ Identifier string } }
	}
	if e = json.Unmarshal(data, &res); e != nil {
		*events = append(*events, "archive JSON: "+e.Error())
		return
	}
	count := 0
	for _, doc := range res.Response.Docs {
		if count >= 2 {
			break
		}
		id := doc.Identifier
		if len(id) < 3 || len(id) > 120 || strings.ContainsAny(id, "/%?#\\ ") {
			continue
		}
		data, e = fetch("https://archive.org/metadata/"+id, 2<<20)
		if e != nil {
			continue
		}
		var item struct {
			Metadata struct {
				Identifier string
				Mediatype  string
				Licenseurl any
				Is_dark    any
			}
			Files []struct{ Name, Format string }
		}
		if json.Unmarshal(data, &item) != nil || item.Metadata.Identifier != id || item.Metadata.Mediatype != "movies" || item.Metadata.Is_dark == true {
			continue
		}
		license := ""
		switch v := item.Metadata.Licenseurl.(type) {
		case string:
			license = v
		case []any:
			if len(v) > 0 {
				license, _ = v[0].(string)
			}
		}
		if !strings.Contains(strings.ToLower(license), "creativecommons.org/publicdomain/") {
			continue
		}
		for _, f := range item.Files {
			if f.Format != "Archive BitTorrent" || !strings.HasSuffix(f.Name, "_archive.torrent") || strings.Contains(f.Name, "/") {
				continue
			}
			link := "https://archive.org/download/" + url.PathEscape(id) + "/" + url.PathEscape(f.Name)
			if e = admit(out, entries, "internet-archive:prelinger", id, license, link, "movie"); e == nil {
				count++
			}
			break
		}
	}
	if count == 0 {
		*events = append(*events, "archive: no individually public-domain-labeled real torrent admitted this bounded run")
	}
}
func main() {
	out := flag.String("out", "", "runner-local evidence directory")
	minimum := flag.Int("minimum", 2, "minimum real torrent identities")
	flag.Parse()
	if *out == "" {
		panic("-out required")
	}
	if e := os.MkdirAll(filepath.Join(*out, "metainfo"), 0755); e != nil {
		panic(e)
	}
	entries := []map[string]any{}
	events := []string{}
	for _, name := range []string{"ubuntu-26.04.1-live-server-amd64.iso.torrent", "ubuntu-26.04.1-desktop-amd64.iso.torrent"} {
		link := "https://releases.ubuntu.com/26.04/" + name
		if e := admit(*out, &entries, "ubuntu:official", name, "official Linux distribution; rights not adjudicated", link, "software"); e != nil {
			events = append(events, "ubuntu "+name+": "+e.Error())
		}
	}
	// A second independent publisher family: Debian official distributed mirror, version-pinned. No ISO payload.
	debianRecord := "debian-13.7.0-amd64-netinst.iso.torrent"
	debianLink := "https://mirrors.edge.kernel.org/debian-cd/current/amd64/bt-cd/" + debianRecord
	if e := admit(*out, &entries, "debian:official-mirror", debianRecord, "official Debian signed distribution; metainfo rights not independently adjudicated", debianLink, "software"); e != nil {
		events = append(events, "debian "+debianRecord+": "+e.Error())
	}
	collectArchive(*out, &entries, &events)
	sort.Slice(entries, func(i, j int) bool { return entries[i]["infohash"].(string) < entries[j]["infohash"].(string) })
	manifest := map[string]any{"schema_version": "cpe-rdte-real-metainfo-v1", "authority_status": "research_source_observations_not_accepted_CPE", "entries": entries, "source_events": events, "no_payload_download": true, "no_swarm_or_dht": true}
	raw, e := json.MarshalIndent(manifest, "", "  ")
	if e != nil {
		panic(e)
	}
	if e = os.WriteFile(filepath.Join(*out, "manifest.json"), append(raw, '\n'), 0644); e != nil {
		panic(e)
	}
	f, e := os.Create(filepath.Join(*out, "bitmagnet-import.ndjson"))
	if e != nil {
		panic(e)
	}
	encoder := json.NewEncoder(f)
	for _, entry := range entries {
		_ = encoder.Encode(map[string]any{"source": entry["source"], "infoHash": entry["infohash"], "name": entry["name"], "size": entry["size_bytes"], "contentType": entry["content_type"]})
		fmt.Printf("CPE-RDTE-REAL: source=%s infohash=%s metainfo_sha256=%s\n", entry["source"], entry["infohash"], entry["metainfo_sha256"])
	}
	if e = f.Close(); e != nil {
		panic(e)
	}
	for _, event := range events {
		fmt.Fprintln(os.Stderr, "CPE-RDTE-SOURCE-EVENT:", event)
	}
	fmt.Printf("CPE-RDTE-REAL: authentic_metainfo_count=%d payload_downloads=0 p2p_connections=0 cpe_promotions=0\n", len(entries))
	if len(entries) < *minimum {
		os.Exit(1)
	}
}

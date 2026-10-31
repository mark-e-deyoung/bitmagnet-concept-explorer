package main

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"
)

func TestValidateMetainfoRedirect(t *testing.T) {
	cases := []struct {
		name  string
		url   string
		via   int
		allow bool
	}{
		{"ubuntu_HTTPS", "https://releases.ubuntu.com/26.04/filename.torrent", 1, true},
		{"debian_mirror_HTTPS", "https://mirrors.edge.kernel.org/debian-cd/current/filename.torrent", 1, true},
		{"debian_secondary_HTTPS", "https://debian.nic.cz/debian-cd/current/filename.torrent", 1, true},
		{"debian_secondary_http_downgrade", "http://debian.nic.cz/debian-cd/current/filename.torrent", 1, false},
		{"archive_cdn_HTTPS", "https://files.archive.org/download/record.torrent", 1, true},
		{"protocol_downgrade", "http://mirrors.edge.kernel.org/debian-cd/current/filename.torrent", 1, false},
		{"foreign_host", "https://attacker.example.org/fake.torrent", 1, false},
		{"spoofed_hostname", "https://mirror.archive.org.attacker.example.org/fake.torrent", 1, false},
		{"redirect_budget", "https://mirrors.edge.kernel.org/debian-cd/current/filename.torrent", 4, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, tt.url, nil)
			if err != nil {
				t.Fatal(err)
			}
			via := make([]*http.Request, tt.via)
			err = validateMetainfoRedirect(req, via)
			if got := err == nil; got != tt.allow {
				t.Fatalf("redirect admission: got allow=%v, want allow=%v (err=%v)", got, tt.allow, err)
			}
		})
	}
}

func TestVerifyPinnedMetainfo(t *testing.T) {
	// Deliberately synthetic v1 metainfo; never confused with genuine publisher data.
	raw := []byte("d4:infod6:lengthi10e4:name4:test6:pieces20:abcdefghijklmnopqrstee")
	hash, _, _, err := decodeMetainfo(raw)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	sha := hex.EncodeToString(sum[:])
	if err := verifyPinnedMetainfo(raw, sha, hash); err != nil {
		t.Fatalf("exact synthetic bytes unexpectedly rejected: %v", err)
	}
	if err := verifyPinnedMetainfo(raw, sha, "0000000000000000000000000000000000000000"); err == nil {
		t.Fatal("wrong pinned raw infohash unexpectedly accepted")
	}
	corrupt := append([]byte(nil), raw...)
	corrupt[12] ^= 1
	if err := verifyPinnedMetainfo(corrupt, sha, hash); err == nil {
		t.Fatal("altered metainfo bytes unexpectedly accepted")
	}
	if err := verifyPinnedMetainfo(raw[:len(raw)-1], sha, hash); err == nil {
		t.Fatal("truncated metainfo unexpectedly accepted")
	}
}

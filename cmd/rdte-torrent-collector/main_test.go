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
		{"arch_publisher_HTTPS", "https://archlinux.org/releng/releases/2026.10.01/torrent/", 1, true},
		{"arch_http_downgrade", "http://archlinux.org/releng/releases/2026.10.01/torrent/", 1, false},
		{"fedora_official_HTTPS", "https://torrent.fedoraproject.org/torrents/Fedora-Workstation-Live-x86_64-44.torrent", 1, true},
		{"fedora_http_downgrade", "http://torrent.fedoraproject.org/torrents/Fedora-Workstation-Live-x86_64-44.torrent", 1, false},
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

func TestPinnedMetainfoDistinguishesWrapperFromInfoIdentity(t *testing.T) {
	// Synthetic, offline bytes. No publisher or file access is required.
	info := "d6:lengthi10e4:name4:test12:piece lengthi16384e6:pieces20:abcdefghijklmnopqrste"
	original := []byte("d7:comment8:original4:info" + info + "e")
	variant := []byte("d7:comment7:variant4:info" + info + "e")
	different := []byte("d7:comment8:original4:infod6:lengthi11e4:name4:test12:piece lengthi16384e6:pieces20:abcdefghijklmnopqrstee")
	v2 := []byte("d4:infod6:lengthi10e12:meta versioni2e4:name4:test6:pieces20:abcdefghijklmnopqrstee")
	hash, _, _, err := decodeMetainfo(original)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(original)
	pin := hex.EncodeToString(sum[:])
	for _, tc := range []struct {
		name string
		raw  []byte
		want string
	}{
		{"original", original, "exact_pinned_bytes"},
		{"wrapper", variant, "same_infohash_different_wrapper"},
		{"other info", different, "different_infohash"},
		{"truncated", original[:len(original)-1], "unresolved"},
		{"unsupported v2 or hybrid", v2, "unresolved"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := classifyPinnedMetainfo(tc.raw, pin, hash)
			if got != tc.want || (err != nil) != (tc.want == "unresolved") {
				t.Fatalf("classification %s err=%v want %s", got, err, tc.want)
			}
			if (verifyPinnedMetainfo(tc.raw, pin, hash) == nil) != (tc.want == "exact_pinned_bytes") {
				t.Fatal("strict original publisher replay gate weakened")
			}
		})
	}
	if state, err := classifyPinnedMetainfo(original, "", hash); err == nil || state != "unresolved" {
		t.Fatal("missing prior full-file digest admitted")
	}
}

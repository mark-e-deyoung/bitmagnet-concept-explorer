package main

import (
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

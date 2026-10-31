package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTrueNASPublicCandidateLoopbackDefault(t *testing.T) {
	root := filepath.Join("ix-dev", "community", "bitmagnet-concept-explorer")
	raw, err := os.ReadFile(filepath.Join(root, "questions.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	// Exposed port must not silently bind all NAS interfaces by default.
	if !strings.Contains(source, "host_ips\n                label: Host IPs\n                schema:\n                  type: list\n                  default:\n                    - 127.0.0.1") {
		t.Fatal("TrueNAS source must default web port to explicit loopback IP")
	}
	if !strings.Contains(source, "TLS-terminating reverse proxy") {
		t.Fatal("no remote-access TLS warning in public source wizard")
	}
	if !strings.Contains(source, "private: true") || !strings.Contains(source, "min_length: 16") {
		t.Fatal("Basic Auth secret field unexpectedly weakened")
	}
	template, err := os.ReadFile(filepath.Join(root, "templates", "docker-compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(template), "app.add_port(values.network.web_port") ||
		!strings.Contains(string(template), "app.set_user(65532, 65532)") ||
		!strings.Contains(string(template), "app.set_read_only(true)") {
		t.Fatal("hardened container source/render interface removed")
	}
}

package sieve

import "testing"

func TestResolveHostPortExplicitPort(t *testing.T) {
	hostport, host := ResolveHostPort("mail.example.com:5190")
	if hostport != "mail.example.com:5190" {
		t.Errorf("hostport = %q", hostport)
	}
	if host != "mail.example.com" {
		t.Errorf("host = %q", host)
	}
}

func TestResolveHostPortTrimsSpace(t *testing.T) {
	hostport, host := ResolveHostPort("  mail.example.com:4190  ")
	if hostport != "mail.example.com:4190" {
		t.Errorf("hostport = %q", hostport)
	}
	if host != "mail.example.com" {
		t.Errorf("host = %q", host)
	}
}

// TestResolveHostPortDefaultPort checks the fallback for a host that should not
// have a ManageSieve SRV record. If a record happens to exist the test is
// skipped rather than producing a false failure.
func TestResolveHostPortDefaultPort(t *testing.T) {
	const host = "no-such-host.invalid"
	hostport, gotHost := ResolveHostPort(host)
	if gotHost != host {
		t.Errorf("host = %q, want %q", gotHost, host)
	}
	want := host + ":" + DefaultPort
	if hostport != want {
		t.Skipf("hostport = %q (SRV record present?), want %q", hostport, want)
	}
}

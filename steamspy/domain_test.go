package steamspy

import (
	"testing"
)

// These tests are offline: they exercise the URI driver's pure string functions.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "steamspy" {
		t.Errorf("Scheme = %q, want steamspy", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "steamspy" {
		t.Errorf("Identity.Binary = %q, want steamspy", info.Identity.Binary)
	}
}

func TestClassifyNumeric(t *testing.T) {
	typ, id, err := Domain{}.Classify("570")
	if err != nil {
		t.Fatalf("Classify error: %v", err)
	}
	if typ != "app" {
		t.Errorf("typ = %q, want app", typ)
	}
	if id != "570" {
		t.Errorf("id = %q, want 570", id)
	}
}

func TestClassifyEmpty(t *testing.T) {
	_, _, err := Domain{}.Classify("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("app", "570")
	if err != nil {
		t.Fatalf("Locate error: %v", err)
	}
	want := "https://store.steampowered.com/app/570/"
	if got != want {
		t.Errorf("Locate = %q, want %q", got, want)
	}
}

func TestLocateUnknownType(t *testing.T) {
	_, err := Domain{}.Locate("unknown", "570")
	if err == nil {
		t.Error("expected error for unknown resource type")
	}
}

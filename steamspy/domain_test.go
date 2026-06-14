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

func TestToAppFree(t *testing.T) {
	w := wireApp{
		AppID:          570,
		Name:           "Dota 2",
		Developer:      "Valve",
		Positive:       1000000,
		Negative:       100000,
		Owners:         "100,000,000 .. 200,000,000",
		AverageForever: 120, // 2 hours
		Price:          "0",
		Genre:          "Action,Free to Play",
		Tags:           map[string]int{"MOBA": 50000, "Free to Play": 40000, "Strategy": 30000},
	}
	a := toApp(w)
	if a.Price != "Free" {
		t.Errorf("Price = %q, want Free", a.Price)
	}
	if a.AvgHours != "2.0" {
		t.Errorf("AvgHours = %q, want 2.0", a.AvgHours)
	}
	if a.AppID != 570 {
		t.Errorf("AppID = %d, want 570", a.AppID)
	}
}

func TestToAppPaid(t *testing.T) {
	w := wireApp{
		AppID:          440,
		Name:           "Team Fortress 2",
		Price:          "999",
		AverageForever: 60,
	}
	a := toApp(w)
	if a.Price != "$9.99" {
		t.Errorf("Price = %q, want $9.99", a.Price)
	}
	if a.AvgHours != "1.0" {
		t.Errorf("AvgHours = %q, want 1.0", a.AvgHours)
	}
}

func TestTopNTags(t *testing.T) {
	tags := map[string]int{
		"A": 100,
		"B": 90,
		"C": 80,
		"D": 70,
		"E": 60,
		"F": 50,
	}
	result := topNTags(tags, 5)
	// Should contain first 5 by count, not F
	for _, want := range []string{"A", "B", "C", "D", "E"} {
		if !containsStr(result, want) {
			t.Errorf("top 5 tags should include %q, got %q", want, result)
		}
	}
	if containsStr(result, "F") {
		t.Errorf("top 5 tags should not include F, got %q", result)
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && (s[:len(sub)] == sub || containsStr(s[1:], sub)))
}

package steamspy_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tamnd/steamspy-cli/steamspy"
)

const fakeAppJSON = `{
  "appid": 570,
  "name": "Dota 2",
  "developer": "Valve",
  "publisher": "Valve",
  "score_rank": "",
  "positive": 1234567,
  "negative": 234567,
  "owners": "100,000,000 .. 200,000,000",
  "average_forever": 11234,
  "average_2weeks": 234,
  "price": "0",
  "ccu": 987654,
  "languages": "English, French",
  "genre": "Action,Free to Play",
  "tags": {"Free to Play": 80000, "MOBA": 70000, "Strategy": 60000, "Multiplayer": 50000, "Co-op": 40000, "PvP": 30000}
}`

const fakeGameMapJSON = `{
  "730": {
    "appid": 730,
    "name": "Counter-Strike 2",
    "developer": "Valve",
    "publisher": "Valve",
    "positive": 500000,
    "negative": 100000,
    "owners": "50,000,000 .. 100,000,000",
    "average_forever": 5000,
    "average_2weeks": 300,
    "price": "0",
    "ccu": 500000,
    "genre": "Action,Free to Play",
    "tags": {"FPS": 90000, "Shooter": 80000}
  },
  "570": {
    "appid": 570,
    "name": "Dota 2",
    "developer": "Valve",
    "publisher": "Valve",
    "positive": 1234567,
    "negative": 234567,
    "owners": "100,000,000 .. 200,000,000",
    "average_forever": 11234,
    "average_2weeks": 234,
    "price": "0",
    "ccu": 987654,
    "genre": "Action,Free to Play",
    "tags": {"MOBA": 70000, "Free to Play": 80000}
  }
}`

func newTestClient(ts *httptest.Server) *steamspy.Client {
	cfg := steamspy.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return steamspy.NewClient(cfg)
}

func TestAppSendsUserAgent(t *testing.T) {
	var gotUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = fmt.Fprint(w, fakeAppJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.App(context.Background(), 570)
	if err != nil {
		t.Fatal(err)
	}
	if gotUA == "" {
		t.Error("User-Agent not sent")
	}
}

func TestAppParsesGame(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeAppJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	a, err := c.App(context.Background(), 570)
	if err != nil {
		t.Fatal(err)
	}
	if a.AppID != 570 {
		t.Errorf("AppID = %d, want 570", a.AppID)
	}
	if a.Name != "Dota 2" {
		t.Errorf("Name = %q, want Dota 2", a.Name)
	}
	if a.Developer != "Valve" {
		t.Errorf("Developer = %q, want Valve", a.Developer)
	}
	if a.Positive != 1234567 {
		t.Errorf("Positive = %d, want 1234567", a.Positive)
	}
	if a.Owners != "100,000,000 .. 200,000,000" {
		t.Errorf("Owners = %q", a.Owners)
	}
	if a.Price != "Free" {
		t.Errorf("Price = %q, want Free", a.Price)
	}
	// average_forever=11234 minutes / 60 = 187.2333... -> "187.2"
	if a.AvgHours != "187.2" {
		t.Errorf("AvgHours = %q, want 187.2", a.AvgHours)
	}
	if a.TopTags == "" {
		t.Error("TopTags should not be empty")
	}
}

func TestAppRetriesOn503(t *testing.T) {
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = fmt.Fprint(w, fakeAppJSON)
	}))
	defer ts.Close()

	cfg := steamspy.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 3
	c := steamspy.NewClient(cfg)

	_, err := c.App(context.Background(), 570)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}

func TestTopSortsAndLimits(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "top100in2weeks") {
			t.Errorf("query %q does not contain top100in2weeks", r.URL.RawQuery)
		}
		_, _ = fmt.Fprint(w, fakeGameMapJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	apps, err := c.Top(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	// limit=1, should get only the highest Positive (570 with 1234567)
	if len(apps) != 1 {
		t.Fatalf("len = %d, want 1", len(apps))
	}
	if apps[0].AppID != 570 {
		t.Errorf("top game AppID = %d, want 570 (highest positive)", apps[0].AppID)
	}
}

func TestTopDefaultLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeGameMapJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	// limit=0 means no limit (returns all)
	apps, err := c.Top(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 2 {
		t.Fatalf("len = %d, want 2", len(apps))
	}
}

func TestGenreParsesGames(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "genre=Action") {
			t.Errorf("query %q does not contain genre=Action", r.URL.RawQuery)
		}
		_, _ = fmt.Fprint(w, fakeGameMapJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	apps, err := c.Genre(context.Background(), "Action", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 2 {
		t.Fatalf("len(apps) = %d, want 2", len(apps))
	}
	// sorted by Positive desc: 570 first (1234567) then 730 (500000)
	if apps[0].AppID != 570 {
		t.Errorf("apps[0].AppID = %d, want 570", apps[0].AppID)
	}
	if apps[1].AppID != 730 {
		t.Errorf("apps[1].AppID = %d, want 730", apps[1].AppID)
	}
}

func TestGenreLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeGameMapJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	apps, err := c.Genre(context.Background(), "Action", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 {
		t.Fatalf("len = %d, want 1 (limit applied)", len(apps))
	}
}

func TestSearchSendsTermParam(t *testing.T) {
	var gotQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = fmt.Fprint(w, fakeGameMapJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Search(context.Background(), "portal", 10)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotQuery, "request=search") {
		t.Errorf("query %q does not contain request=search", gotQuery)
	}
	if !strings.Contains(gotQuery, "term=portal") {
		t.Errorf("query %q does not contain term=portal", gotQuery)
	}
}

func TestSearchLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeGameMapJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	apps, err := c.Search(context.Background(), "valve", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 {
		t.Fatalf("len = %d, want 1 (limit applied)", len(apps))
	}
}

func TestAppPaidPrice(t *testing.T) {
	const paidJSON = `{
		"appid": 292030,
		"name": "The Witcher 3: Wild Hunt",
		"developer": "CD PROJEKT RED",
		"publisher": "CD PROJEKT RED",
		"positive": 600000,
		"negative": 10000,
		"owners": "20,000,000 .. 50,000,000",
		"average_forever": 3000,
		"average_2weeks": 0,
		"price": "3999",
		"genre": "RPG",
		"tags": {"RPG": 50000}
	}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, paidJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	a, err := c.App(context.Background(), 292030)
	if err != nil {
		t.Fatal(err)
	}
	if a.Price != "$39.99" {
		t.Errorf("Price = %q, want $39.99", a.Price)
	}
	// 3000 / 60 = 50.0
	if a.AvgHours != "50.0" {
		t.Errorf("AvgHours = %q, want 50.0", a.AvgHours)
	}
}

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
  "languages": "English, French"
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
    "ccu": 500000
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
    "ccu": 987654
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
	g, err := c.App(context.Background(), 570)
	if err != nil {
		t.Fatal(err)
	}
	if g.AppID != 570 {
		t.Errorf("AppID = %d, want 570", g.AppID)
	}
	if g.Name != "Dota 2" {
		t.Errorf("Name = %q, want Dota 2", g.Name)
	}
	if g.Developer != "Valve" {
		t.Errorf("Developer = %q, want Valve", g.Developer)
	}
	if g.Positive != 1234567 {
		t.Errorf("Positive = %d, want 1234567", g.Positive)
	}
	if g.Owners != "100,000,000 .. 200,000,000" {
		t.Errorf("Owners = %q", g.Owners)
	}
	if g.Price != "0" {
		t.Errorf("Price = %q, want 0", g.Price)
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

func TestGenreParsesGames(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeGameMapJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	games, err := c.Genre(context.Background(), "Action", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(games) != 2 {
		t.Fatalf("len(games) = %d, want 2", len(games))
	}
	// sorted by AppID ascending: 570 then 730
	if games[0].AppID != 570 {
		t.Errorf("games[0].AppID = %d, want 570", games[0].AppID)
	}
	if games[1].AppID != 730 {
		t.Errorf("games[1].AppID = %d, want 730", games[1].AppID)
	}
}

func TestGenrePageParam(t *testing.T) {
	var gotQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = fmt.Fprint(w, fakeGameMapJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Genre(context.Background(), "Action", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotQuery, "page=1") {
		t.Errorf("query %q does not contain page=1", gotQuery)
	}
}

func TestTagParsesGames(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeGameMapJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	games, err := c.Tag(context.Background(), "Multiplayer", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(games) != 2 {
		t.Fatalf("len(games) = %d, want 2", len(games))
	}
}

func TestTopParsesGames2Weeks(t *testing.T) {
	var gotQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = fmt.Fprint(w, fakeGameMapJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	games, err := c.Top(context.Background(), "2weeks")
	if err != nil {
		t.Fatal(err)
	}
	if len(games) != 2 {
		t.Fatalf("len(games) = %d, want 2", len(games))
	}
	if !strings.Contains(gotQuery, "top100in2weeks") {
		t.Errorf("query %q does not contain top100in2weeks", gotQuery)
	}
}

func TestTopParsesGamesForever(t *testing.T) {
	var gotQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = fmt.Fprint(w, fakeGameMapJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Top(context.Background(), "forever")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotQuery, "top100forever") {
		t.Errorf("query %q does not contain top100forever", gotQuery)
	}
}

func TestTopParsesGamesOwned(t *testing.T) {
	var gotQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = fmt.Fprint(w, fakeGameMapJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Top(context.Background(), "owned")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotQuery, "top100owned") {
		t.Errorf("query %q does not contain top100owned", gotQuery)
	}
}

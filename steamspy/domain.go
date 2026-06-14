// Package steamspy exposes SteamSpy as a kit Domain driver.
// A multi-domain host (ant) enables it with a single blank import:
//
//	import _ "github.com/tamnd/steamspy-cli/steamspy"
//
// The same Domain also builds the standalone steamspy binary (see cli/root.go),
// so the binary and a host share one source of truth.
package steamspy

import (
	"context"
	"fmt"
	"strconv"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

func init() { kit.Register(Domain{}) }

// Domain is the steamspy driver.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against,
// and the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "steamspy",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "steamspy",
			Short:  "SteamSpy -- Steam game ownership and player stats",
			Long: `steamspy fetches game details, genre lists, and top-100 charts
from steamspy.com. No API key required.`,
			Site: Host,
			Repo: "https://github.com/tamnd/steamspy-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{
		Name:    "app",
		Group:   "read",
		Single:  true,
		Summary: "Get game details by Steam app ID",
		Args:    []kit.Arg{{Name: "appid", Help: "Steam application ID"}},
	}, appOp)

	kit.Handle(app, kit.OpMeta{
		Name:    "top",
		Group:   "read",
		List:    true,
		Summary: "List top 100 games by 2-week players",
	}, topOp)

	kit.Handle(app, kit.OpMeta{
		Name:    "genre",
		Group:   "read",
		List:    true,
		Summary: "List games by genre",
		Args:    []kit.Arg{{Name: "genre", Help: "genre name (e.g. Action, RPG, Strategy)"}},
	}, genreOp)

	kit.Handle(app, kit.OpMeta{
		Name:    "search",
		Group:   "read",
		List:    true,
		Summary: "Search games by name",
		Args:    []kit.Arg{{Name: "term", Help: "partial name to search for"}},
	}, searchOp)
}

// newClient builds the client from host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

// --- inputs ---

type appInput struct {
	AppID  string  `kit:"arg" help:"Steam application ID"`
	Client *Client `kit:"inject"`
}

type topInput struct {
	Limit  int     `kit:"flag,inherit" help:"maximum number of results (default 20)"`
	Client *Client `kit:"inject"`
}

type genreInput struct {
	Genre  string  `kit:"arg"          help:"genre name (e.g. Action, RPG, Strategy)"`
	Limit  int     `kit:"flag,inherit" help:"maximum number of results (default 20)"`
	Client *Client `kit:"inject"`
}

type searchInput struct {
	Term   string  `kit:"arg"          help:"partial name to search for"`
	Limit  int     `kit:"flag,inherit" help:"maximum number of results (default 10)"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func appOp(ctx context.Context, in appInput, emit func(App) error) error {
	id, err := strconv.Atoi(in.AppID)
	if err != nil {
		return errs.Usage("appid must be an integer: %v", err)
	}
	a, err := in.Client.App(ctx, id)
	if err != nil {
		return mapErr(err)
	}
	return emit(*a)
}

func topOp(ctx context.Context, in topInput, emit func(App) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	apps, err := in.Client.Top(ctx, limit)
	if err != nil {
		return mapErr(err)
	}
	for _, a := range apps {
		if err := emit(a); err != nil {
			return err
		}
	}
	return nil
}

func genreOp(ctx context.Context, in genreInput, emit func(App) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	apps, err := in.Client.Genre(ctx, in.Genre, limit)
	if err != nil {
		return mapErr(err)
	}
	for _, a := range apps {
		if err := emit(a); err != nil {
			return err
		}
	}
	return nil
}

func searchOp(ctx context.Context, in searchInput, emit func(App) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 10
	}
	apps, err := in.Client.Search(ctx, in.Term, limit)
	if err != nil {
		return mapErr(err)
	}
	for _, a := range apps {
		if err := emit(a); err != nil {
			return err
		}
	}
	return nil
}

// --- Resolver ---

// Classify turns an input into the canonical (type, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	if input == "" {
		return "", "", errs.Usage("empty steamspy reference")
	}
	// Numeric input -> app ID
	if _, err := strconv.Atoi(input); err == nil {
		return "app", input, nil
	}
	return "app", input, nil
}

// Locate returns the live https URL for a (type, id).
func (Domain) Locate(t, id string) (string, error) {
	switch t {
	case "app":
		return fmt.Sprintf("https://store.steampowered.com/app/%s/", id), nil
	default:
		return "", errs.Usage("steamspy has no resource type %q", t)
	}
}

// mapErr converts a library error into the kit error kind.
func mapErr(err error) error {
	return err
}

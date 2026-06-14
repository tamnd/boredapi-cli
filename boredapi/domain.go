package boredapi

import (
	"context"
	"strings"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes boredapi as a kit Domain: a driver that a multi-domain
// host (ant) enables with a single blank import,
//
//	import _ "github.com/tamnd/boredapi-cli/boredapi"
//
// exactly as a database/sql program enables a driver with `import _
// "github.com/lib/pq"`. The init below registers it; the host then dereferences
// boredapi:// URIs by routing to the operations Register installs. The same
// Domain also builds the standalone boredapi binary (see cli.NewApp), so the
// binary and a host share one source of truth.
func init() { kit.Register(Domain{}) }

// Domain is the boredapi kit driver. It carries no state; the per-run
// client is built by the factory Register hands to kit.
type Domain struct{}

// Info describes the scheme, hostnames, and the identity used in help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "boredapi",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "boredapi",
			Short:  "Suggest random activities when bored via the Bored API",
			Long: `Suggest random activities when bored via the Bored API.

boredapi reads public data from the Bored API over plain HTTPS, shapes it into
clean records, and prints output that pipes into the rest of your tools. No API
key, nothing to run alongside it.

Get a random activity, filter by type, number of participants, or look up a
specific activity by its key.`,
			Site: Host,
			Repo: "https://github.com/tamnd/boredapi-cli",
		},
	}
}

// Register installs the client factory and all operations onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{Name: "activity", Group: "read", Single: true,
		Summary: "Get a random activity suggestion"}, activityOp)
}

// newClient builds the client from the host-resolved config.
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

// --- input structs ---

type activityInput struct {
	Type         string  `kit:"flag" help:"filter by type (education/recreational/social/diy/charity/cooking/relaxation/music/busywork)"`
	Participants int     `kit:"flag" help:"number of participants (0 = any)"`
	Key          string  `kit:"flag" help:"get a specific activity by key"`
	Client       *Client `kit:"inject"`
}

// --- handlers ---

func activityOp(ctx context.Context, in activityInput, emit func(*Activity) error) error {
	a, err := in.Client.GetActivity(ctx, ActivityFilter{
		Type:         in.Type,
		Participants: in.Participants,
		Key:          in.Key,
	})
	if err != nil {
		return err
	}
	return emit(a)
}

// Classify turns a reference into (type, id). A key is a numeric string.
func (Domain) Classify(input string) (string, string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", "", errs.Usage("empty boredapi reference")
	}
	return "activity", input, nil
}

// Locate returns the live API URL for a (type, id) pair.
func (Domain) Locate(t, id string) (string, error) {
	if t != "activity" {
		return "", errs.Usage("boredapi has no resource type %q", t)
	}
	return BaseURL + "/api/activity?key=" + id, nil
}

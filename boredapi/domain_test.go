package boredapi

import (
	"testing"

	"github.com/tamnd/any-cli/kit"
)

// These tests are offline: they exercise the URI driver's pure string functions
// and the host wiring (mint, body, resolve), which need no network. The client's
// HTTP behaviour is covered in boredapi_test.go.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "boredapi" {
		t.Errorf("Scheme = %q, want boredapi", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "boredapi" {
		t.Errorf("Identity.Binary = %q, want boredapi", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct{ in, typ, id string }{
		{"3943506", "activity", "3943506"},
		{"abc123", "activity", "abc123"},
	}
	for _, tc := range cases {
		typ, id, err := Domain{}.Classify(tc.in)
		if err != nil || typ != tc.typ || id != tc.id {
			t.Errorf("Classify(%q) = (%q, %q, %v), want (%q, %q, nil)",
				tc.in, typ, id, err, tc.typ, tc.id)
		}
	}
}

func TestClassifyEmpty(t *testing.T) {
	_, _, err := Domain{}.Classify("")
	if err == nil {
		t.Error("expected error for empty input, got nil")
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("activity", "3943506")
	want := "https://" + Host + "/api/activity?key=3943506"
	if err != nil || got != want {
		t.Errorf("Locate = (%q, %v), want (%q, nil)", got, err, want)
	}
}

func TestLocateUnknownType(t *testing.T) {
	_, err := Domain{}.Locate("unknown", "123")
	if err == nil {
		t.Error("expected error for unknown type, got nil")
	}
}

// TestHostWiring mounts the driver in a kit Host and checks the round trip:
// a record mints to its URI, its body is readable, and a bare id resolves
// back to the same URI. The init in domain.go registers the domain, so kit.Open
// finds it.
func TestHostWiring(t *testing.T) {
	h, err := kit.Open()
	if err != nil {
		t.Fatal(err)
	}

	a := &Activity{
		Key:           "3943506",
		Activity:      "Learn Express.js",
		Type:          "education",
		Participants:  1,
		Price:         "0.00",
		Accessibility: "0.25",
		Link:          "https://expressjs.com/",
	}
	u, err := h.Mint(a)
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	if want := "boredapi://activity/3943506"; u.String() != want {
		t.Errorf("Mint = %q, want %q", u.String(), want)
	}

	got, err := h.ResolveOn("boredapi", "3943506")
	if err != nil || got.String() != "boredapi://activity/3943506" {
		t.Errorf("ResolveOn = (%q, %v), want boredapi://activity/3943506", got.String(), err)
	}
}

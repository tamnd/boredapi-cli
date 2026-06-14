package boredapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func newTestClient(baseURL string) *Client {
	cfg := DefaultConfig()
	cfg.BaseURL = baseURL
	cfg.Rate = 0
	cfg.Retries = 0
	return NewClient(cfg)
}

func TestGetActivity_Random(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		if r.URL.Path != "/api/activity" {
			t.Errorf("path = %q, want /api/activity", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(wireActivity{
			Activity:      "Learn Express.js",
			Type:          "education",
			Participants:  1,
			Price:         0.0,
			Link:          "https://expressjs.com/",
			Key:           "3943506",
			Accessibility: 0.25,
		})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	a, err := c.GetActivity(context.Background(), ActivityFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if a.Key != "3943506" {
		t.Errorf("Key = %q, want 3943506", a.Key)
	}
	if a.Activity != "Learn Express.js" {
		t.Errorf("Activity = %q, want Learn Express.js", a.Activity)
	}
	if a.Type != "education" {
		t.Errorf("Type = %q, want education", a.Type)
	}
	if a.Price != "0.00" {
		t.Errorf("Price = %q, want 0.00", a.Price)
	}
	if a.Accessibility != "0.25" {
		t.Errorf("Accessibility = %q, want 0.25", a.Accessibility)
	}
}

func TestGetActivity_TypeFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("type") != "recreational" {
			t.Errorf("type param = %q, want recreational", q.Get("type"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(wireActivity{
			Activity:      "Go for a walk",
			Type:          "recreational",
			Participants:  1,
			Price:         0.0,
			Key:           "1234567",
			Accessibility: 0.1,
		})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	a, err := c.GetActivity(context.Background(), ActivityFilter{Type: "recreational"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Type != "recreational" {
		t.Errorf("Type = %q, want recreational", a.Type)
	}
}

func TestGetActivity_ParticipantsFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("participants") != "2" {
			t.Errorf("participants param = %q, want 2", q.Get("participants"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(wireActivity{
			Activity:      "Play chess",
			Type:          "social",
			Participants:  2,
			Price:         0.0,
			Key:           "7654321",
			Accessibility: 0.2,
		})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	a, err := c.GetActivity(context.Background(), ActivityFilter{Participants: 2})
	if err != nil {
		t.Fatal(err)
	}
	if a.Participants != 2 {
		t.Errorf("Participants = %d, want 2", a.Participants)
	}
}

func TestGetActivity_KeyFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("key") != "3943506" {
			t.Errorf("key param = %q, want 3943506", q.Get("key"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(wireActivity{
			Activity:      "Learn Express.js",
			Type:          "education",
			Participants:  1,
			Price:         0.0,
			Link:          "https://expressjs.com/",
			Key:           "3943506",
			Accessibility: 0.25,
		})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	a, err := c.GetActivity(context.Background(), ActivityFilter{Key: "3943506"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Key != "3943506" {
		t.Errorf("Key = %q, want 3943506", a.Key)
	}
}

func TestGetActivity_NoMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(wireActivity{
			Error: "No activity found with the specified parameters",
		})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.GetActivity(context.Background(), ActivityFilter{Type: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for no-match response, got nil")
	}
	if err.Error() != "No activity found with the specified parameters" {
		t.Errorf("err = %q, want the API error message", err.Error())
	}
}

func TestBuildQuery(t *testing.T) {
	cases := []struct {
		f    ActivityFilter
		want url.Values
	}{
		{ActivityFilter{}, url.Values{}},
		{ActivityFilter{Type: "education"}, url.Values{"type": {"education"}}},
		{ActivityFilter{Participants: 3}, url.Values{"participants": {"3"}}},
		{ActivityFilter{Key: "abc"}, url.Values{"key": {"abc"}}},
		{ActivityFilter{Type: "social", Participants: 2, Key: "xyz"}, url.Values{"type": {"social"}, "participants": {"2"}, "key": {"xyz"}}},
	}
	for _, tc := range cases {
		q := buildQuery(tc.f)
		got, err := url.ParseQuery(q)
		if err != nil {
			t.Fatalf("ParseQuery(%q): %v", q, err)
		}
		for k, vs := range tc.want {
			if got.Get(k) != vs[0] {
				t.Errorf("filter %+v: param %q = %q, want %q", tc.f, k, got.Get(k), vs[0])
			}
		}
		// no extra params
		for k := range got {
			if _, ok := tc.want[k]; !ok {
				t.Errorf("filter %+v: unexpected param %q", tc.f, k)
			}
		}
	}
}

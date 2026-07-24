package yahoo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPageOptionsSuffix(t *testing.T) {
	cases := []struct {
		opts PageOptions
		want string
	}{
		{PageOptions{}, ""},
		{PageOptions{Start: 0, Count: 25}, ";start=0;count=25"},
		{PageOptions{Start: 50, Count: 25}, ";start=50;count=25"},
		{PageOptions{Start: 50, Count: 0}, ";start=50"},
		{PageOptions{Start: -5, Count: 10}, ";start=0;count=10"},
	}
	for _, c := range cases {
		if got := c.opts.suffix(); got != c.want {
			t.Errorf("%+v.suffix() = %q, want %q", c.opts, got, c.want)
		}
	}
}

// The paged transactions call must include the ;start;count segment in the URL;
// the default call must omit it.
func TestTransactionsPaginationEndpoint(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`{"fantasy_content":{"league":{"transactions":[]}}}`))
	}))
	defer srv.Close()

	c, err := NewClient(WithTokens("t", ""), WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatalf("construction failed: %v", err)
	}
	ctx := context.Background()

	if _, err := c.GetLeagueTransactionsPage(ctx, "123.l.456", PageOptions{Start: 25, Count: 25}); err != nil {
		t.Fatalf("paged: %v", err)
	}
	if want := "league/123.l.456/transactions;start=25;count=25"; gotPath != "/"+want {
		t.Errorf("paged path = %q, want to contain %q", gotPath, want)
	}

	if _, err := c.GetLeagueTransactions(ctx, "123.l.456"); err != nil {
		t.Fatalf("default: %v", err)
	}
	if want := "league/123.l.456/transactions"; gotPath != "/"+want {
		t.Errorf("default path = %q, want %q (no pagination segment)", gotPath, "/"+want)
	}
}

func TestDraftResultsPaginationEndpoint(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`{"fantasy_content":{"league":{"draft_results":[]}}}`))
	}))
	defer srv.Close()

	c, err := NewClient(WithTokens("t", ""), WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatalf("construction failed: %v", err)
	}

	if _, err := c.GetLeagueDraftResultsPage(context.Background(), "123.l.456", PageOptions{Start: 100, Count: 50}); err != nil {
		t.Fatalf("paged: %v", err)
	}
	if want := "/league/123.l.456/draftresults;start=100;count=50"; gotPath != want {
		t.Errorf("paged path = %q, want %q", gotPath, want)
	}
}

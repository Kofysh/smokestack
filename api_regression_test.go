package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLiveStreamPoolsAreIndependent(t *testing.T) {
	first, second := &API{}, &API{}
	address := "198.51.100.10"
	for i := 0; i < maxLivePerAddr; i++ {
		if !first.streams.take(address) {
			t.Fatalf("first API rejected connection %d", i+1)
		}
	}
	if first.streams.take(address) {
		t.Fatal("first API exceeded the per-address limit")
	}
	if !second.streams.take(address) {
		t.Fatal("a separate API shared the first instance's limit")
	}
	first.streams.done(address)
	if !first.streams.take(address) {
		t.Fatal("a released connection could not be reused")
	}
}

func TestLiveStreamRejectionReturnsRetryAfter(t *testing.T) {
	api := &API{}
	address := "198.51.100.20"
	for i := 0; i < maxLivePerAddr; i++ {
		if !api.streams.take(address) {
			t.Fatal("unexpected connection limit")
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/live", nil)
	req.RemoteAddr = address + ":1234"
	rec := httptest.NewRecorder()
	api.live(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("HTTP %d, expected 503", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got != "10" {
		t.Errorf("Retry-After = %q, expected 10", got)
	}
}

func TestIngestRejectsUnauthorizedAndOversizedBatches(t *testing.T) {
	api := &API{}
	call := func(body string, token bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/ingest", strings.NewReader(body))
		if token {
			req.Header.Set("Authorization", "Bearer test-token")
		}
		rec := httptest.NewRecorder()
		api.ingest(rec, req)
		return rec
	}
	if got := call("{", false).Code; got != http.StatusUnauthorized {
		t.Fatalf("malformed unauthenticated request returned HTTP %d", got)
	}
	batch := fmt.Sprintf("{\"probe\":\"remote\",\"batch\":[%s{}]}", strings.Repeat("{},", maxIngestBatch))
	if got := call(batch, true).Code; got != http.StatusRequestEntityTooLarge {
		t.Fatalf("batch over %d measurements returned HTTP %d", maxIngestBatch, got)
	}
}

func TestPublicSeriesKeepsSharedCache(t *testing.T) {
	store, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	category, err := store.CreateCategory("cache-test", "Cache", "Cache", true)
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.CreateTarget(&Target{
		CategoryID: category, Slug: "shared-cache", Title: "Shared cache", Host: "192.0.2.1",
		Proto: "icmp", IntervalS: 60, Packets: 10, SpacingMs: 100,
		TimeoutMs: 1000, Public: true, Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	api := &API{store: store, token: "secret", probeID: 1}
	for _, authenticated := range []bool{false, true} {
		name := "anonymous"
		if authenticated {
			name = "authenticated"
		}
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/series?target=%d&from=-3h", id), nil)
			if authenticated {
				req.Header.Set("Authorization", "Bearer secret")
			}
			rec := httptest.NewRecorder()
			api.series(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("HTTP %d: %s", rec.Code, rec.Body.String())
			}
			if cc := rec.Header().Get("Cache-Control"); !strings.HasPrefix(cc, "public, max-age=") {
				t.Errorf("unexpected cache policy: %q", cc)
			}
			if vary := rec.Header().Get("Vary"); vary != "" {
				t.Errorf("public target should not vary by caller: %q", vary)
			}
		})
	}
}

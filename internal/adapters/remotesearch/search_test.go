package remotesearch

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestSearchParsesGitHubResults(t *testing.T) {
	original := httpClient
	httpClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if !strings.Contains(request.URL.RawQuery, "topic%3Aagent-skills") {
			t.Fatalf("unexpected GitHub query: %s", request.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"total_count":1,"items":[{"name":"demo-skill","full_name":"acme/demo-skill"}]}`)),
			Header:     make(http.Header),
		}, nil
	})}
	t.Cleanup(func() { httpClient = original })

	items, err := Search("demo", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(items) != 1 || items[0].Name != "demo-skill" {
		t.Fatalf("Search() = %#v, want demo-skill", items)
	}
}

package auth

import (
	"net/http"
	"testing"
)

func TestApplyOpenAIAuthHeaders(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, TokenURL, nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	ApplyOpenAIAuthHeaders(req)

	if got := req.Header.Get("Accept"); got != "application/json" {
		t.Fatalf("Accept = %q", got)
	}
	if got := req.Header.Get("User-Agent"); got != OpenAICodexCLIUserAgent {
		t.Fatalf("User-Agent = %q", got)
	}
}

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestValidateResponsesAPIRequestRejectsUnsupportedModel(t *testing.T) {
	result := ValidateResponsesAPIRequest(
		[]byte(`{"model":"gpt-unknown","input":"hello"}`),
		[]string{"gpt-5.5", "gpt-5.4"},
	)

	if result.Valid {
		t.Fatal("expected unsupported model to be invalid")
	}
	if len(result.Errors) != 1 {
		t.Fatalf("errors length = %d, want 1", len(result.Errors))
	}
	if got := result.Errors[0].Code; got != "unsupported_model" {
		t.Fatalf("error code = %q, want unsupported_model", got)
	}
}

func TestSendListIncludesOptionalHasMore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	SendList(c, "list", []string{"gpt-5.5"}, true)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body struct {
		Object  string   `json:"object"`
		Data    []string `json:"data"`
		HasMore *bool    `json:"has_more"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Object != "list" {
		t.Fatalf("object = %q, want list", body.Object)
	}
	if len(body.Data) != 1 || body.Data[0] != "gpt-5.5" {
		t.Fatalf("data = %#v, want [gpt-5.5]", body.Data)
	}
	if body.HasMore == nil || !*body.HasMore {
		t.Fatalf("has_more = %#v, want true", body.HasMore)
	}
}

func TestHTTPStatusCodeForCommonAPIErrors(t *testing.T) {
	tests := []struct {
		name string
		code ErrorCode
		want int
	}{
		{name: "auth", code: ErrCodeInvalidAPIKey, want: http.StatusUnauthorized},
		{name: "rate limit", code: ErrCodeRateLimitReached, want: http.StatusTooManyRequests},
		{name: "unsupported model", code: ErrCodeUnsupportedModel, want: http.StatusBadRequest},
		{name: "upstream timeout", code: ErrCodeUpstreamTimeout, want: http.StatusInternalServerError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := HTTPStatusCode(test.code); got != test.want {
				t.Fatalf("HTTPStatusCode(%q) = %d, want %d", test.code, got, test.want)
			}
		})
	}
}

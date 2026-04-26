package wsrelay

import (
	"net/http"
	"testing"

	"github.com/codex2api/auth"
	"github.com/codex2api/proxy"
)

func TestPrepareWebsocketHeadersUsesConfiguredDefaultsAndBetaFeatures(t *testing.T) {
	exec := NewExecutor()
	cfg := &proxy.DeviceProfileConfig{
		UserAgent:              "codex_cli_rs/0.120.0 (Mac OS 15.5.0; arm64) Apple_Terminal/464",
		PackageVersion:         "0.120.0",
		RuntimeVersion:         "0.120.0",
		OS:                     "MacOS",
		Arch:                   "arm64",
		StabilizeDeviceProfile: true,
		BetaFeatures:           "multi_agent",
	}
	ginHeaders := http.Header{
		"Originator": []string{"custom-originator"},
	}

	account := &auth.Account{DBID: 42, AccountID: "42"}
	headers := exec.prepareWebsocketHeaders(account, "token-123", "session-123", "api-key-1", cfg, ginHeaders)

	if got := headers.Get("Authorization"); got != "Bearer token-123" {
		t.Fatalf("Authorization = %q", got)
	}
	if got := headers.Get("OpenAI-Beta"); got != responsesWebsocketBetaHeader {
		t.Fatalf("OpenAI-Beta = %q", got)
	}
	if got := headers.Get("X-Codex-Beta-Features"); got != "multi_agent" {
		t.Fatalf("X-Codex-Beta-Features = %q", got)
	}
	if got := headers.Get("User-Agent"); got != auth.OpenAICodexCLIUserAgent {
		t.Fatalf("User-Agent = %q", got)
	}
	if got := headers.Get("Version"); got != auth.OpenAICodexCLIVersion {
		t.Fatalf("Version = %q", got)
	}
	if got := headers.Get("Originator"); got != proxy.Originator {
		t.Fatalf("Originator = %q", got)
	}
	if got := headers.Get("Chatgpt-Account-Id"); got != "42" {
		t.Fatalf("Chatgpt-Account-Id = %q", got)
	}
	if got := headers.Get("Conversation_id"); got != "session-123" {
		t.Fatalf("Conversation_id = %q", got)
	}
}

func TestPrepareWebsocketHeadersUsesConfiguredUserAgentWithoutStabilization(t *testing.T) {
	exec := NewExecutor()
	cfg := &proxy.DeviceProfileConfig{
		UserAgent:      "codex-tui/0.124.0 (Mac OS 26.4.1; arm64) Apple_Terminal/470 (codex-tui; 0.124.0)",
		PackageVersion: "0.124.0",
	}
	ginHeaders := http.Header{
		"User-Agent": []string{"codex_cli_rs/0.130.0 (Mac OS 15.5.0; arm64) Apple_Terminal/464"},
	}

	account := &auth.Account{DBID: 42, AccountID: "42", Type: auth.AccountTypeAccessToken}
	headers := exec.prepareWebsocketHeaders(account, "token-123", "session-123", "", cfg, ginHeaders)

	if got := headers.Get("User-Agent"); got != cfg.UserAgent {
		t.Fatalf("User-Agent = %q", got)
	}
	if got := headers.Get("Version"); got != "0.124.0" {
		t.Fatalf("Version = %q", got)
	}
}

func TestPrepareWebsocketHeadersForcesCodexCLIForOAuthAccount(t *testing.T) {
	exec := NewExecutor()
	cfg := &proxy.DeviceProfileConfig{
		UserAgent:      "codex-tui/0.124.0",
		PackageVersion: "0.124.0",
	}
	account := &auth.Account{
		DBID:         52,
		Platform:     auth.PlatformOpenAI,
		Type:         auth.AccountTypeOAuth,
		RefreshToken: "rt-123",
	}

	headers := exec.prepareWebsocketHeaders(account, "token-123", "session-123", "", cfg, nil)

	if got := headers.Get("User-Agent"); got != auth.OpenAICodexCLIUserAgent {
		t.Fatalf("User-Agent = %q", got)
	}
	if got := headers.Get("Version"); got != auth.OpenAICodexCLIVersion {
		t.Fatalf("Version = %q", got)
	}
	if got := headers.Get("Originator"); got != proxy.Originator {
		t.Fatalf("Originator = %q", got)
	}
}

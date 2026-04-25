package proxy

import "testing"

func TestDeviceProfileConfigFromEnv(t *testing.T) {
	cfg := DeviceProfileConfigFromEnv(func(key string) string {
		switch key {
		case "STABILIZE_DEVICE_PROFILE":
			return "true"
		case "CODEX_USER_AGENT":
			return "codex_cli_rs/0.120.0 (Mac OS 15.5.0; arm64) Apple_Terminal/464"
		case "CODEX_PACKAGE_VERSION":
			return "0.120.0"
		case "CODEX_RUNTIME_VERSION":
			return "0.120.0"
		case "CODEX_OS":
			return "MacOS"
		case "CODEX_ARCH":
			return "arm64"
		case "CODEX_BETA_FEATURES":
			return "multi_agent"
		default:
			return ""
		}
	})

	if cfg == nil {
		t.Fatal("expected config")
	}
	if !cfg.StabilizeDeviceProfile {
		t.Fatal("expected device profile stabilization to be enabled")
	}
	if cfg.UserAgent != "codex_cli_rs/0.120.0 (Mac OS 15.5.0; arm64) Apple_Terminal/464" {
		t.Fatalf("UserAgent = %q", cfg.UserAgent)
	}
	if cfg.PackageVersion != "0.120.0" {
		t.Fatalf("PackageVersion = %q", cfg.PackageVersion)
	}
	if cfg.RuntimeVersion != "0.120.0" {
		t.Fatalf("RuntimeVersion = %q", cfg.RuntimeVersion)
	}
	if cfg.OS != "MacOS" {
		t.Fatalf("OS = %q", cfg.OS)
	}
	if cfg.Arch != "arm64" {
		t.Fatalf("Arch = %q", cfg.Arch)
	}
	if cfg.BetaFeatures != "multi_agent" {
		t.Fatalf("BetaFeatures = %q", cfg.BetaFeatures)
	}
}

func TestConfiguredClientProfileUsesConfiguredUserAgent(t *testing.T) {
	profile, ok := ConfiguredClientProfile(&DeviceProfileConfig{
		UserAgent:      "codex-tui/0.124.0 (Mac OS 26.4.1; arm64) Apple_Terminal/470 (codex-tui; 0.124.0)",
		PackageVersion: "0.124.0",
	})
	if !ok {
		t.Fatal("expected configured profile")
	}
	if profile.UserAgent != "codex-tui/0.124.0 (Mac OS 26.4.1; arm64) Apple_Terminal/470 (codex-tui; 0.124.0)" {
		t.Fatalf("UserAgent = %q", profile.UserAgent)
	}
	if profile.Version != "0.124.0" {
		t.Fatalf("Version = %q", profile.Version)
	}
}

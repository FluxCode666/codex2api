package proxy

import (
	"testing"

	utls "github.com/refraction-networking/utls"
)

func TestBuildNodeJS24ClientHelloSpecAdvertisesHTTP2First(t *testing.T) {
	spec := buildNodeJS24ClientHelloSpec()

	var alpn *utls.ALPNExtension
	for _, ext := range spec.Extensions {
		if candidate, ok := ext.(*utls.ALPNExtension); ok {
			alpn = candidate
			break
		}
	}

	if alpn == nil {
		t.Fatal("expected ALPN extension in client hello spec")
	}

	if len(alpn.AlpnProtocols) < 2 {
		t.Fatalf("expected h2 and http/1.1 ALPN protocols, got %v", alpn.AlpnProtocols)
	}

	if alpn.AlpnProtocols[0] != "h2" {
		t.Fatalf("first ALPN protocol = %q, want %q", alpn.AlpnProtocols[0], "h2")
	}

	if alpn.AlpnProtocols[1] != "http/1.1" {
		t.Fatalf("second ALPN protocol = %q, want %q", alpn.AlpnProtocols[1], "http/1.1")
	}
}

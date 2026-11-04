# setsig-go

Official Go SDK for [setsig.com](https://setsig.com) — Physical Presence & Verification Infrastructure-as-a-Service (PaaS/IaaS).

[![Go Reference](https://pkg.go.dev/badge/github.com/setsig/setsig-go.svg)](https://pkg.go.dev/github.com/setsig/setsig-go)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

Zero 3rd-party dependencies. Built exclusively using the Go standard library (`net/http`, `crypto/hmac`, `crypto/subtle`).

---

## Installation

```bash
go get github.com/setsig/setsig-go
```

Requirements: Go 1.23 or newer.

---

## Quickstart

### 1. Initialize Client

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/setsig/setsig-go"
)

func main() {
	client, err := setsig.NewClient("sk_live_...",
		setsig.WithTimeout(10*time.Second),
		setsig.WithMaxRetries(3),
	)
	if err != nil {
		log.Fatalf("failed to initialize setsig client: %v", err)
	}

	// 1. Generate dynamic QR presence token
	token, err := client.Tokens.Generate(context.Background(), setsig.GenerateTokenRequest{
		VenueID: "ven_ist_headquarters",
		TTL:     60,
		Format:  setsig.TokenFormatSVG,
	})
	if err != nil {
		log.Fatalf("generate token failed: %v", err)
	}

	fmt.Printf("Dynamic Token Generated: %s (Expires: %s)\n", token.DynamicCode, token.ExpiresAt)
}
```

---

### 2. Verify Session (Telemetric Presence & Attestation)

```go
res, err := client.Sessions.Verify(context.Background(), setsig.VerifySessionRequest{
	VenueID:          "ven_ist_headquarters",
	TokenID:          token.TokenID,
	ClientToken:      "device_bound_token_...",
	Latitude:         41.0082,
	Longitude:        28.9784,
	Accuracy:         4.5,
	AttestationToken: "apple_or_google_hardware_attest_token",
	VisualProofToken: "vpt_...",
})
if err != nil {
	switch e := err.(type) {
	case *setsig.ReplayAttackError:
		log.Printf("Security alert: Replay / Double-spend detected: %v", e)
	case *setsig.GeofenceOutOfBoundsError:
		log.Printf("Physical boundary breach: %v", e)
	case *setsig.InsufficientCreditsError:
		log.Printf("Billing alert: Balance depleted: %v", e)
	default:
		log.Printf("Verification failed: %v", err)
	}
	return
}

fmt.Printf("Session %s VERIFIED! Distance: %.2fm, Risk Score: %.4f\n", 
	res.SessionID, res.DistanceMeters, res.RiskScore)
```

---

### 3. Verify Incoming Webhooks (Timing-Safe Constant Time Comparison)

Patent-protected replay resistance and timing attack mitigation:

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/setsig/setsig-go"
)

func main() {
	webhookSecret := "whsec_..."

	http.HandleFunc("/webhooks/setsig", setsig.WebhookHandler(webhookSecret, func(w http.ResponseWriter, r *http.Request, event *setsig.WebhookEvent) {
		fmt.Printf("Received authentic setsig event: %s (ID: %s)\n", event.Event, event.ID)

		switch event.Event {
		case "session.verified":
			// Handle verified session
		case "venue.anomaly_detected":
			// Handle perimeter breach
		}

		w.WriteHeader(http.StatusOK)
	}))

	http.ListenAndServe(":8080", nil)
}
```

---

## Core Features

- **Zero External Dependencies:** Built entirely with Go standard library.
- **Timing-Safe Cryptography:** Webhook HMAC verification utilizes `crypto/subtle.ConstantTimeCompare`.
- **Automatic Retries & Exponential Backoff:** Gracefully handles 429 Rate Limits and 5xx network hiccups with jitter.
- **Idempotency Keys:** Native support for preventing duplicate token charges via `setsig.RequestOptions{IdempotencyKey: "..."}`.
- **Strong Typed Error Hierarchy:** Programmatically inspect `AuthenticationError`, `ReplayAttackError`, `GeofenceOutOfBoundsError`, `InsufficientCreditsError`, and `RateLimitError`.

---

## License

Apache-2.0 License. Commercial verification engine intellectual property held by AVCI Ventures.


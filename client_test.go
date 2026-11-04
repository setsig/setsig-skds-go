package setsig_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/setsig/setsig-go"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newMockClient(t *testing.T, handler http.HandlerFunc, opts ...setsig.Option) *setsig.Client {
	t.Helper()
	mockHTTP := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			handler(rec, req)
			return rec.Result(), nil
		}),
	}
	allOpts := append([]setsig.Option{
		setsig.WithBaseURL("https://api.setsig.test"),
		setsig.WithHTTPClient(mockHTTP),
	}, opts...)

	client, err := setsig.NewClient("test_key", allOpts...)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return client
}

func TestNewClient(t *testing.T) {
	_, err := setsig.NewClient("")
	if err != setsig.ErrMissingAPIKey {
		t.Fatalf("expected ErrMissingAPIKey, got: %v", err)
	}

	client, err := setsig.NewClient("test_api_key_valid",
		setsig.WithBaseURL("https://custom.api.setsig.com/"),
		setsig.WithTimeout(5*time.Second),
		setsig.WithMaxRetries(2),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestTokensService_Generate(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/tokens/generate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test_key" {
			t.Errorf("missing or invalid auth header")
		}
		if r.Header.Get("Idempotency-Key") != "idemp_123" {
			t.Errorf("missing idempotency key: %s", r.Header.Get("Idempotency-Key"))
		}

		var req setsig.GenerateTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode error: %v", err)
		}

		resp := setsig.APIResponseEnvelope[setsig.GenerateTokenResponse]{
			Data: setsig.GenerateTokenResponse{
				TokenID:     "tok_abc123",
				VenueID:     req.VenueID,
				DynamicCode: "884920",
				ExpiresAt:   time.Now().Add(60 * time.Second),
				Payload:     "setsig://v1/t/tok_abc123?c=884920",
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}

	client := newMockClient(t, handler)

	res, err := client.Tokens.Generate(context.Background(), setsig.GenerateTokenRequest{
		VenueID: "ven_venue_1",
		TTL:     60,
		Format:  setsig.TokenFormatRaw,
	}, setsig.RequestOptions{IdempotencyKey: "idemp_123"})

	if err != nil {
		t.Fatalf("generate token failed: %v", err)
	}

	if res.TokenID != "tok_abc123" || res.DynamicCode != "884920" {
		t.Fatalf("unexpected token response: %+v", res)
	}
}

func TestSessionsService_Verify(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sessions/verify" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := setsig.APIResponseEnvelope[setsig.VerifySessionResponse]{
			Data: setsig.VerifySessionResponse{
				SessionID:      "ses_verified_1",
				TenantID:       "ten_alpha",
				VenueID:        "ven_venue_1",
				Status:         "VERIFIED",
				VerifiedAt:     time.Now(),
				DistanceMeters: 4.2,
				RiskScore:      0.02,
				VisualProof: &setsig.VisualProofSummary{
					Status:        "PASSED",
					LivenessScore: 0.99,
				},
				ClaimTokens: []string{"clm_1", "clm_2"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}

	client := newMockClient(t, handler)

	res, err := client.Sessions.Verify(context.Background(), setsig.VerifySessionRequest{
		VenueID:     "ven_venue_1",
		TokenID:     "tok_123",
		ClientToken: "client_device_token_abc",
		Latitude:    41.0082,
		Longitude:   28.9784,
		Accuracy:    5.0,
	})

	if err != nil {
		t.Fatalf("verify session failed: %v", err)
	}

	if res.Status != "VERIFIED" || res.VisualProof == nil || res.VisualProof.LivenessScore < 0.9 {
		t.Fatalf("unexpected session response: %+v", res)
	}
}

func TestVenuesService_CRUD(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/venues":
			_ = json.NewEncoder(w).Encode(setsig.APIResponseEnvelope[setsig.Venue]{
				Data: setsig.Venue{
					ID:           "ven_created_1",
					Name:         "Istanbul Hub",
					RadiusMeters: 25.0,
					Status:       "ACTIVE",
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/venues/ven_created_1":
			_ = json.NewEncoder(w).Encode(setsig.APIResponseEnvelope[setsig.Venue]{
				Data: setsig.Venue{
					ID:           "ven_created_1",
					Name:         "Istanbul Hub",
					RadiusMeters: 25.0,
					Status:       "ACTIVE",
				},
			})
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/venues/ven_created_1":
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}

	client := newMockClient(t, handler)

	created, err := client.Venues.Create(context.Background(), setsig.CreateVenueRequest{
		Name:         "Istanbul Hub",
		Latitude:     41.0082,
		Longitude:    28.9784,
		RadiusMeters: 25.0,
	})
	if err != nil {
		t.Fatalf("create venue failed: %v", err)
	}
	if created.ID != "ven_created_1" {
		t.Fatalf("expected ven_created_1, got: %s", created.ID)
	}

	getVenue, err := client.Venues.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get venue failed: %v", err)
	}
	if getVenue.Name != "Istanbul Hub" {
		t.Fatalf("unexpected venue name: %s", getVenue.Name)
	}

	err = client.Venues.Delete(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("delete venue failed: %v", err)
	}
}

func TestBillingService(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/billing/balance":
			_ = json.NewEncoder(w).Encode(setsig.APIResponseEnvelope[setsig.BillingBalance]{
				Data: setsig.BillingBalance{
					BalanceCredits: 25000,
					Currency:       "USD",
					Tier:           "ENTERPRISE",
					AutoTopup:      true,
				},
			})
		case "/v1/billing/estimate":
			_ = json.NewEncoder(w).Encode(setsig.APIResponseEnvelope[setsig.EstimateCostResponse]{
				Data: setsig.EstimateCostResponse{
					UnitCostCredits:  12,
					TotalCostCredits: 1200,
					GeoMultiplier:    1.2,
					Currency:         "USD",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}

	client := newMockClient(t, handler)

	balance, err := client.Billing.GetBalance(context.Background())
	if err != nil {
		t.Fatalf("get balance failed: %v", err)
	}
	if balance.BalanceCredits != 25000 {
		t.Fatalf("unexpected balance: %d", balance.BalanceCredits)
	}

	estimate, err := client.Billing.EstimateCost(context.Background(), setsig.EstimateCostRequest{
		CountryCode:       "TR",
		StrictAttestation: true,
		StrictVisualProof: true,
		Quantity:          100,
	})
	if err != nil {
		t.Fatalf("estimate failed: %v", err)
	}
	if estimate.TotalCostCredits != 1200 {
		t.Fatalf("unexpected cost: %d", estimate.TotalCostCredits)
	}
}

func TestErrorMappings(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		checkErr   func(err error) bool
	}{
		{
			name:       "401 Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"error":"Invalid API key","code":"UNAUTHORIZED"}`,
			checkErr: func(err error) bool {
				var authErr *setsig.AuthenticationError
				return isAs(err, &authErr)
			},
		},
		{
			name:       "402 Insufficient Credits",
			statusCode: http.StatusPaymentRequired,
			body:       `{"error":"Insufficient credits","code":"INSUFFICIENT_CREDITS"}`,
			checkErr: func(err error) bool {
				var creditErr *setsig.InsufficientCreditsError
				return isAs(err, &creditErr)
			},
		},
		{
			name:       "409 Replay Attack",
			statusCode: http.StatusConflict,
			body:       `{"error":"Replay attack detected","code":"DOUBLE_SPEND_REPLAY_ATTACK"}`,
			checkErr: func(err error) bool {
				var replayErr *setsig.ReplayAttackError
				return isAs(err, &replayErr)
			},
		},
		{
			name:       "422 Geofence Out Of Bounds",
			statusCode: http.StatusUnprocessableEntity,
			body:       `{"error":"Distance exceeds venue geofence limit","code":"GEOFENCE_BREACH"}`,
			checkErr: func(err error) bool {
				var geoErr *setsig.GeofenceOutOfBoundsError
				return isAs(err, &geoErr)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}

			client := newMockClient(t, handler, setsig.WithMaxRetries(0))

			_, err := client.Tokens.Generate(context.Background(), setsig.GenerateTokenRequest{VenueID: "v1"})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.checkErr(err) {
				t.Fatalf("error was not of expected type: %v", err)
			}
		})
	}
}

func TestRetries_OnServerError(t *testing.T) {
	var attempts int32

	handler := func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&attempts, 1)
		if current < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"temporarily unavailable"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(setsig.APIResponseEnvelope[setsig.GenerateTokenResponse]{
			Data: setsig.GenerateTokenResponse{TokenID: "tok_recovered"},
		})
	}

	client := newMockClient(t, handler, setsig.WithMaxRetries(3))

	res, err := client.Tokens.Generate(context.Background(), setsig.GenerateTokenRequest{VenueID: "v1"})
	if err != nil {
		t.Fatalf("expected successful recovery after retry, got: %v", err)
	}
	if res.TokenID != "tok_recovered" {
		t.Fatalf("unexpected token id: %s", res.TokenID)
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Fatalf("expected 3 attempts, got: %d", attempts)
	}
}

func isAs[T error](err error, target *T) bool {
	if val, ok := err.(T); ok {
		*target = val
		return true
	}
	return false
}

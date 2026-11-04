package setsig

import (
	"encoding/json"
	"time"
)

// TokenFormat defines the visual output format for a generated dynamic token.
type TokenFormat string

const (
	TokenFormatRaw     TokenFormat = "raw"
	TokenFormatDataURL TokenFormat = "data_url"
	TokenFormatSVG     TokenFormat = "svg"
)

// GenerateTokenRequest contains parameters to generate a dynamic presence QR token.
type GenerateTokenRequest struct {
	VenueID  string            `json:"venue_id"`
	TTL      int               `json:"ttl,omitempty"`      // Duration in seconds (default: 60)
	Format   TokenFormat       `json:"format,omitempty"`   // Format: raw, data_url, svg
	Metadata map[string]string `json:"metadata,omitempty"` // Arbitrary client context metadata
}

// GenerateTokenResponse contains the generated token payload and visual renderings.
type GenerateTokenResponse struct {
	TokenID     string    `json:"token_id"`
	VenueID     string    `json:"venue_id"`
	DynamicCode string    `json:"dynamic_code"`
	ExpiresAt   time.Time `json:"expires_at"`
	QRDataURL   string    `json:"qr_data_url,omitempty"`
	QRSVG       string    `json:"qr_svg,omitempty"`
	Payload     string    `json:"payload"`
}

// VerifySessionRequest specifies the client telemetry and tokens for physical verification.
type VerifySessionRequest struct {
	VenueID          string   `json:"venue_id"`
	TokenID          string   `json:"token_id"`
	ClientToken      string   `json:"client_token"`
	Latitude         float64  `json:"latitude"`
	Longitude        float64  `json:"longitude"`
	Accuracy         float64  `json:"accuracy"`
	Altitude         *float64 `json:"altitude,omitempty"`
	Speed            *float64 `json:"speed,omitempty"`
	Heading          *float64 `json:"heading,omitempty"`
	AttestationToken string   `json:"attestation_token,omitempty"`
	VisualProofToken string   `json:"visual_proof_token,omitempty"`
	ClientIP         string   `json:"client_ip,omitempty"`
}

// VisualProofSummary encapsulates forensic visual verification results.
type VisualProofSummary struct {
	Status            string    `json:"status"`
	StorageKey        string    `json:"storage_key,omitempty"`
	LivenessScore     float64   `json:"liveness_score"`
	VisualTamperScore float64   `json:"visual_tamper_score"`
	CapturedAt        time.Time `json:"captured_at"`
	AnalysisNotes     []string  `json:"analysis_notes,omitempty"`
}

// VerifySessionResponse represents the outcome of physical presence validation.
type VerifySessionResponse struct {
	SessionID      string              `json:"session_id"`
	TenantID       string              `json:"tenant_id"`
	VenueID        string              `json:"venue_id"`
	Status         string              `json:"status"` // VERIFIED, REJECTED, REVOKED
	VerifiedAt     time.Time           `json:"verified_at"`
	DistanceMeters float64             `json:"distance_meters"`
	RiskScore      float64             `json:"risk_score"`
	VisualProof    *VisualProofSummary `json:"visual_proof,omitempty"`
	ClaimTokens    []string            `json:"claim_tokens,omitempty"`
}

// Venue represents a physical perimeter enrolled in setsig.
type Venue struct {
	ID                string    `json:"id"`
	TenantID          string    `json:"tenant_id"`
	Name              string    `json:"name"`
	Latitude          float64   `json:"latitude"`
	Longitude         float64   `json:"longitude"`
	RadiusMeters      float64   `json:"radius_meters"`
	StrictAttestation bool      `json:"strict_attestation"`
	StrictVisualProof bool      `json:"strict_visual_proof"`
	DynamicSeedSecret string    `json:"dynamic_seed_secret,omitempty"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// CreateVenueRequest holds parameters to enroll a new venue.
type CreateVenueRequest struct {
	Name              string  `json:"name"`
	Latitude          float64 `json:"latitude"`
	Longitude         float64 `json:"longitude"`
	RadiusMeters      float64 `json:"radius_meters"`
	StrictAttestation bool    `json:"strict_attestation"`
	StrictVisualProof bool    `json:"strict_visual_proof"`
}

// UpdateVenueRequest allows partial updates to an existing venue.
type UpdateVenueRequest struct {
	Name              *string  `json:"name,omitempty"`
	RadiusMeters      *float64 `json:"radius_meters,omitempty"`
	StrictAttestation *bool    `json:"strict_attestation,omitempty"`
	StrictVisualProof *bool    `json:"strict_visual_proof,omitempty"`
}

// ListVenuesOptions provides pagination for listing venues.
type ListVenuesOptions struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// ListVenuesResponse represents the paginated list of venues.
type ListVenuesResponse struct {
	Venues []Venue `json:"venues"`
	Total  int     `json:"total"`
}

// BillingBalance provides current credits and tier info.
type BillingBalance struct {
	BalanceCredits int64  `json:"balance_credits"`
	Currency       string `json:"currency"`
	Tier           string `json:"tier"`
	AutoTopup      bool   `json:"auto_topup"`
}

// EstimateCostRequest computes dynamic credit consumption.
type EstimateCostRequest struct {
	CountryCode       string `json:"country_code"`
	StrictAttestation bool   `json:"strict_attestation"`
	StrictVisualProof bool   `json:"strict_visual_proof"`
	Quantity          int    `json:"quantity"`
}

// EstimateCostResponse provides tiered calculation results.
type EstimateCostResponse struct {
	UnitCostCredits  int64   `json:"unit_cost_credits"`
	TotalCostCredits int64   `json:"total_cost_credits"`
	GeoMultiplier    float64 `json:"geo_multiplier"`
	Currency         string  `json:"currency"`
}

// WebhookEvent represents an incoming webhook payload delivered by setsig infrastructure.
type WebhookEvent struct {
	ID        string          `json:"id"`
	Event     string          `json:"event"`
	Timestamp int64           `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// APIResponseEnvelope is standard JSON wrapper returned by the backend.
type APIResponseEnvelope[T any] struct {
	Data    T      `json:"data"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
	Code    string `json:"code,omitempty"`
}

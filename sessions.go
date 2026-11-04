package setsig

import (
	"context"
	"fmt"
	"net/http"
)

// SessionsService coordinates physical presence verification and audit inspection.
type SessionsService struct {
	client *Client
}

// Verify verifies physical presence with telemetry, dynamic QR TOTP, and hardware attestation.
func (s *SessionsService) Verify(ctx context.Context, req VerifySessionRequest, reqOpts ...RequestOptions) (*VerifySessionResponse, error) {
	var envelope APIResponseEnvelope[VerifySessionResponse]
	err := s.client.doRequest(ctx, http.MethodPost, "/v1/sessions/verify", req, &envelope, reqOpts...)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

// Get retrieves details of an append-only verified session record.
func (s *SessionsService) Get(ctx context.Context, sessionID string) (*VerifySessionResponse, error) {
	path := fmt.Sprintf("/v1/sessions/%s", sessionID)
	var envelope APIResponseEnvelope[VerifySessionResponse]
	err := s.client.doRequest(ctx, http.MethodGet, path, nil, &envelope)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}


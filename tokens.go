package setsig

import (
	"context"
	"net/http"
)

// TokensService handles dynamic TOTP QR generation for presence verification.
type TokensService struct {
	client *Client
}

// Generate requests a new dynamic cryptographic presence token for a venue.
func (s *TokensService) Generate(ctx context.Context, req GenerateTokenRequest, reqOpts ...RequestOptions) (*GenerateTokenResponse, error) {
	var envelope APIResponseEnvelope[GenerateTokenResponse]
	err := s.client.doRequest(ctx, http.MethodPost, "/v1/tokens/generate", req, &envelope, reqOpts...)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}


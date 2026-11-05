package setsig

import (
	"context"
	"net/http"
)

// TerminalsService provides methods for kiosk displays and Gatekeeper turnstile sessions.
type TerminalsService struct {
	client *Client
}

// CreateSession establishes a scoped session token for a kiosk tablet or turnstile display.
func (s *TerminalsService) CreateSession(ctx context.Context, req TerminalSessionRequest) (*TerminalSessionResponse, error) {
	var resp TerminalSessionResponse
	err := s.client.doRequest(ctx, http.MethodPost, "/v1/terminals/session", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}


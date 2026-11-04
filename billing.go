package setsig

import (
	"context"
	"net/http"
)

// BillingService manages tenant credits and cost estimation.
type BillingService struct {
	client *Client
}

// GetBalance returns the current credit balance and account tier.
func (s *BillingService) GetBalance(ctx context.Context) (*BillingBalance, error) {
	var envelope APIResponseEnvelope[BillingBalance]
	err := s.client.doRequest(ctx, http.MethodGet, "/v1/billing/balance", nil, &envelope)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

// EstimateCost calculates dynamic credit consumption taking geolocational multipliers and features into account.
func (s *BillingService) EstimateCost(ctx context.Context, req EstimateCostRequest) (*EstimateCostResponse, error) {
	var envelope APIResponseEnvelope[EstimateCostResponse]
	err := s.client.doRequest(ctx, http.MethodPost, "/v1/billing/estimate", req, &envelope)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}


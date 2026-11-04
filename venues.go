package setsig

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// VenuesService provides methods for managing geofenced perimeters and venue configurations.
type VenuesService struct {
	client *Client
}

// Create enrolls a new physical venue perimeter with cryptographic seed secret.
func (s *VenuesService) Create(ctx context.Context, req CreateVenueRequest) (*Venue, error) {
	var envelope APIResponseEnvelope[Venue]
	err := s.client.doRequest(ctx, http.MethodPost, "/v1/venues", req, &envelope)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

// Get fetches details and settings for a specific venue by its ID.
func (s *VenuesService) Get(ctx context.Context, id string) (*Venue, error) {
	path := fmt.Sprintf("/v1/venues/%s", id)
	var envelope APIResponseEnvelope[Venue]
	err := s.client.doRequest(ctx, http.MethodGet, path, nil, &envelope)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

// List returns a paginated list of venues enrolled under the current tenant.
func (s *VenuesService) List(ctx context.Context, opts *ListVenuesOptions) (*ListVenuesResponse, error) {
	path := "/v1/venues"
	if opts != nil {
		q := url.Values{}
		if opts.Limit > 0 {
			q.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Offset > 0 {
			q.Set("offset", strconv.Itoa(opts.Offset))
		}
		if encoded := q.Encode(); encoded != "" {
			path = fmt.Sprintf("%s?%s", path, encoded)
		}
	}

	var envelope APIResponseEnvelope[ListVenuesResponse]
	err := s.client.doRequest(ctx, http.MethodGet, path, nil, &envelope)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

// Update modifies venue parameters such as name, radius, or enforcement flags.
func (s *VenuesService) Update(ctx context.Context, id string, req UpdateVenueRequest) (*Venue, error) {
	path := fmt.Sprintf("/v1/venues/%s", id)
	var envelope APIResponseEnvelope[Venue]
	err := s.client.doRequest(ctx, http.MethodPatch, path, req, &envelope)
	if err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

// Delete archives or deactivates a venue.
func (s *VenuesService) Delete(ctx context.Context, id string) error {
	path := fmt.Sprintf("/v1/venues/%s", id)
	return s.client.doRequest(ctx, http.MethodDelete, path, nil, nil)
}


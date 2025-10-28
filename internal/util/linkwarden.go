package util

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
)

var linkwardenHTTPClient = &http.Client{}

type LinkwardenRequest struct {
	URL      string
	Endpoint string
	APIToken string
	Method   string
	Body     any
	OKCodes  []int
}

type LinkwardenResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

func DoLinkwardenRequest(ctx context.Context, req LinkwardenRequest) (*LinkwardenResponse, error) {
	method := http.MethodGet
	if req.Method != "" {
		method = req.Method
	}

	var reqData []byte
	if req.Body != nil {
		if d, err := json.Marshal(req.Body); err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		} else {
			reqData = d
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, req.URL+req.Endpoint, bytes.NewBuffer(reqData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+req.APIToken)

	resp, err := linkwardenHTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	okCodes := req.OKCodes
	if len(okCodes) == 0 {
		okCodes = []int{http.StatusOK}
	}

	if !slices.Contains(okCodes, resp.StatusCode) {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &LinkwardenResponse{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       data,
	}, nil
}

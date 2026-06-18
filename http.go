package xpaymentsdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type httpClient struct {
	baseURL string
	client  *http.Client
}

type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type paginationRequest struct {
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
	Filters  []filterCondition `json:"filters,omitempty"`
}

type filterCondition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
}

func newHTTPClient(baseURL string, client *http.Client) *httpClient {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &httpClient{baseURL: baseURL, client: client}
}

func (c *httpClient) createPaymentOrder(ctx context.Context, req CreatePaymentOrderRequest) (*PaymentOrder, error) {
	var out PaymentOrder
	if err := c.do(ctx, http.MethodPost, "/internal/payment/orders", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *httpClient) getPaymentOrder(ctx context.Context, id uint64) (*PaymentOrder, error) {
	var out PaymentOrder
	if err := c.do(ctx, http.MethodGet, "/internal/payment/orders/"+strconv.FormatUint(id, 10), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *httpClient) getPaymentOrderByMerchantOrder(ctx context.Context, req GetPaymentOrderByMerchantOrderRequest) (*PaymentOrder, error) {
	var out PaymentOrder
	if err := c.do(ctx, http.MethodPost, "/internal/payment/orders/by-merchant-order", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *httpClient) queryPaymentOrder(ctx context.Context, id uint64) (*PaymentOrder, error) {
	var out PaymentOrder
	if err := c.do(ctx, http.MethodPost, "/internal/payment/orders/"+strconv.FormatUint(id, 10)+"/query", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *httpClient) cancelPaymentOrder(ctx context.Context, id uint64) (*PaymentOrder, error) {
	var out PaymentOrder
	if err := c.do(ctx, http.MethodPost, "/internal/payment/orders/"+strconv.FormatUint(id, 10)+"/cancel", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *httpClient) listPaymentOrders(ctx context.Context, req ListPaymentOrdersRequest) (*ListPaymentOrdersResponse, error) {
	body := paginationRequest{Page: req.Page, PageSize: req.PageSize}
	appendFilter := func(field string, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		body.Filters = append(body.Filters, filterCondition{Field: field, Operator: "=", Value: value})
	}
	appendFilter("merchantCode", req.MerchantCode)
	appendFilter("tenantId", req.TenantID)
	appendFilter("userId", req.UserID)
	appendFilter("status", req.Status)
	var out ListPaymentOrdersResponse
	if err := c.do(ctx, http.MethodPost, "/internal/payment/orders/list", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *httpClient) listSupportedCurrencies(ctx context.Context) (*ListSupportedCurrenciesResponse, error) {
	var out ListSupportedCurrenciesResponse
	if err := c.do(ctx, http.MethodGet, "/internal/payment/currencies", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *httpClient) listAvailablePaymentChannels(ctx context.Context, req ListAvailablePaymentChannelsRequest) (*ListAvailablePaymentChannelsResponse, error) {
	var items []AvailablePaymentChannel
	if err := c.do(ctx, http.MethodPost, "/internal/payment/channels/available", req, &items); err != nil {
		return nil, err
	}
	return &ListAvailablePaymentChannelsResponse{Items: items}, nil
}

func (c *httpClient) do(ctx context.Context, method string, path string, body any, out any) error {
	if c == nil || c.baseURL == "" {
		return fmt.Errorf("xpayment http fallback is not configured")
	}
	endpoint, err := joinURL(c.baseURL, path)
	if err != nil {
		return err
	}
	var reader io.Reader
	if body != nil {
		payload, marshalErr := json.Marshal(body)
		if marshalErr != nil {
			return fmt.Errorf("marshal xpayment http request: %w", marshalErr)
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(raw))}
	}
	var wrapped envelope
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return fmt.Errorf("decode xpayment http envelope: %w", err)
	}
	if wrapped.Code != http.StatusOK {
		return &HTTPError{Code: wrapped.Code, Message: wrapped.Message, StatusCode: resp.StatusCode}
	}
	if out == nil || len(wrapped.Data) == 0 {
		return nil
	}
	if err := json.Unmarshal(wrapped.Data, out); err != nil {
		return fmt.Errorf("decode xpayment http data: %w", err)
	}
	return nil
}

func joinURL(baseURL string, pathValue string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/" + strings.TrimLeft(pathValue, "/")
	return parsed.String(), nil
}

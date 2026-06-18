package xpaymentsdk

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"foundation/integration/grpcx"
	pb "xpayment-svc/proto"
)

type client struct {
	cfg     Config
	managed *grpcx.ManagedClient
	grpc    pb.PaymentServiceClient
	http    *httpClient
	timeout time.Duration
}

func NewClient(cfg Config) (Client, error) {
	cfg.GRPCAddress = strings.TrimSpace(cfg.GRPCAddress)
	cfg.HTTPBaseURL = strings.TrimRight(strings.TrimSpace(cfg.HTTPBaseURL), "/")
	if cfg.GRPCAddress == "" && cfg.HTTPBaseURL == "" {
		return nil, fmt.Errorf("xpayment sdk requires grpc address or http base url")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	c := &client{
		cfg:     cfg,
		http:    newHTTPClient(cfg.HTTPBaseURL, &http.Client{Timeout: timeout}),
		timeout: timeout,
	}
	if cfg.GRPCAddress != "" {
		managed, err := grpcx.NewManagedClient(grpcx.ClientConfig{
			Enabled:          true,
			Address:          cfg.GRPCAddress,
			TimeoutSecs:      int(timeout.Seconds()),
			ServiceName:      firstNonEmpty(cfg.ServiceName, "xpayment_sdk"),
			TargetName:       "xpayment_svc",
			MonitorEnabled:   cfg.GRPCMonitor.Enabled,
			AlertEnabled:     cfg.GRPCAlert.Enabled,
			Reconnect:        cfg.GRPCReconnect,
			FallbackTelegram: cfg.FallbackTelegram,
		})
		if err != nil {
			return nil, err
		}
		c.managed = managed
		if managed != nil && managed.Conn() != nil {
			c.grpc = pb.NewPaymentServiceClient(managed.Conn())
			managed.StartMonitor(context.Background())
		}
	}
	return c, nil
}

func (c *client) CreatePaymentOrder(ctx context.Context, req CreatePaymentOrderRequest) (*PaymentOrder, error) {
	if c.useHTTP() {
		return c.http.createPaymentOrder(ctx, req)
	}
	protoReq, err := createToProto(req)
	if err != nil {
		return nil, err
	}
	callCtx, cancel := c.withTimeout(ctx)
	resp, err := c.grpc.CreatePaymentOrder(callCtx, protoReq)
	cancel()
	if err == nil {
		return orderFromProto(resp)
	}
	if shouldFallback(err) {
		return c.http.createPaymentOrder(ctx, req)
	}
	return nil, err
}

func (c *client) GetPaymentOrder(ctx context.Context, id uint64) (*PaymentOrder, error) {
	if c.useHTTP() {
		return c.http.getPaymentOrder(ctx, id)
	}
	callCtx, cancel := c.withTimeout(ctx)
	resp, err := c.grpc.GetPaymentOrder(callCtx, &pb.GetPaymentOrderRequest{Id: id})
	cancel()
	if err == nil {
		return orderFromProto(resp)
	}
	if shouldFallback(err) {
		return c.http.getPaymentOrder(ctx, id)
	}
	return nil, err
}

func (c *client) GetPaymentOrderByMerchantOrder(ctx context.Context, req GetPaymentOrderByMerchantOrderRequest) (*PaymentOrder, error) {
	if c.useHTTP() {
		return c.http.getPaymentOrderByMerchantOrder(ctx, req)
	}
	callCtx, cancel := c.withTimeout(ctx)
	resp, err := c.grpc.GetPaymentOrderByMerchantOrder(callCtx, &pb.GetPaymentOrderByMerchantOrderRequest{
		MerchantCode: req.MerchantCode, MerchantOrderId: req.MerchantOrderID,
	})
	cancel()
	if err == nil {
		return orderFromProto(resp)
	}
	if shouldFallback(err) {
		return c.http.getPaymentOrderByMerchantOrder(ctx, req)
	}
	return nil, err
}

func (c *client) QueryPaymentOrder(ctx context.Context, id uint64) (*PaymentOrder, error) {
	if c.useHTTP() {
		return c.http.queryPaymentOrder(ctx, id)
	}
	callCtx, cancel := c.withTimeout(ctx)
	resp, err := c.grpc.QueryPaymentOrder(callCtx, &pb.GetPaymentOrderRequest{Id: id})
	cancel()
	if err == nil {
		return orderFromProto(resp)
	}
	if shouldFallback(err) {
		return c.http.queryPaymentOrder(ctx, id)
	}
	return nil, err
}

func (c *client) CancelPaymentOrder(ctx context.Context, id uint64) (*PaymentOrder, error) {
	if c.useHTTP() {
		return c.http.cancelPaymentOrder(ctx, id)
	}
	callCtx, cancel := c.withTimeout(ctx)
	resp, err := c.grpc.CancelPaymentOrder(callCtx, &pb.GetPaymentOrderRequest{Id: id})
	cancel()
	if err == nil {
		return orderFromProto(resp)
	}
	if shouldFallback(err) {
		return c.http.cancelPaymentOrder(ctx, id)
	}
	return nil, err
}

func (c *client) ListPaymentOrders(ctx context.Context, req ListPaymentOrdersRequest) (*ListPaymentOrdersResponse, error) {
	if c.useHTTP() {
		return c.http.listPaymentOrders(ctx, req)
	}
	callCtx, cancel := c.withTimeout(ctx)
	resp, err := c.grpc.ListPaymentOrders(callCtx, &pb.ListPaymentOrdersRequest{
		Page: int32(req.Page), PageSize: int32(req.PageSize), MerchantCode: req.MerchantCode, TenantId: req.TenantID, UserId: req.UserID, Status: req.Status,
	})
	cancel()
	if err == nil {
		return listOrdersFromProto(resp)
	}
	if shouldFallback(err) {
		return c.http.listPaymentOrders(ctx, req)
	}
	return nil, err
}

func (c *client) ListSupportedCurrencies(ctx context.Context) (*ListSupportedCurrenciesResponse, error) {
	if c.useHTTP() {
		return c.http.listSupportedCurrencies(ctx)
	}
	callCtx, cancel := c.withTimeout(ctx)
	resp, err := c.grpc.ListSupportedCurrencies(callCtx, &pb.ListSupportedCurrenciesRequest{})
	cancel()
	if err == nil {
		return &ListSupportedCurrenciesResponse{Currencies: resp.GetCurrencies()}, nil
	}
	if shouldFallback(err) {
		return c.http.listSupportedCurrencies(ctx)
	}
	return nil, err
}

func (c *client) ListAvailablePaymentChannels(ctx context.Context, req ListAvailablePaymentChannelsRequest) (*ListAvailablePaymentChannelsResponse, error) {
	if c.useHTTP() {
		return c.http.listAvailablePaymentChannels(ctx, req)
	}
	callCtx, cancel := c.withTimeout(ctx)
	resp, err := c.grpc.ListAvailablePaymentChannels(callCtx, &pb.ListAvailablePaymentChannelsRequest{
		MerchantCode: req.MerchantCode, OrderAmount: req.OrderAmount, OrderCurrency: req.OrderCurrency, PaymentCurrency: req.PaymentCurrency,
	})
	cancel()
	if err == nil {
		return availableChannelsFromProto(resp), nil
	}
	if shouldFallback(err) {
		return c.http.listAvailablePaymentChannels(ctx, req)
	}
	return nil, err
}

func (c *client) Close() error {
	if c == nil || c.managed == nil {
		return nil
	}
	return c.managed.Close()
}

func (c *client) useHTTP() bool {
	return c == nil || c.cfg.PreferHTTP || c.grpc == nil
}

func (c *client) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(ctx, c.timeout)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

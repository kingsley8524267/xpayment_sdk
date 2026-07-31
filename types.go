package xpaymentsdk

import (
	"context"
	"time"

	"foundation/http/security/servicejwt"
	"foundation/integration/grpcx"
	"foundation/runtime/alerts"
	"foundation/runtime/readiness"
)

const defaultTimeout = 5 * time.Second

const (
	OrderStatusPaid             = "paid"
	OrderStatusManuallyApproved = "manually_approved"
	SettlementSourceProvider    = "provider"
	SettlementSourceManual      = "manual"
)

type Config struct {
	Endpoint         grpcx.EndpointConfig
	HTTPBaseURL      string
	Timeout          time.Duration
	ServiceJWT       servicejwt.ClientConfig
	ServiceJWTSigner *servicejwt.SignerRuntime
	Discovery        *grpcx.DiscoveryRuntime
	GRPCMonitor      grpcx.MonitorConfig
	GRPCAlert        grpcx.AlertConfig
	GRPCReconnect    grpcx.ReconnectConfig
	ReadinessMode    string
	Readiness        *readiness.Registry
	Alerts           *alerts.Dispatcher
	PreferHTTP       bool
}

type JSONMap map[string]any

type Client interface {
	CreatePaymentOrder(ctx context.Context, req CreatePaymentOrderRequest) (*PaymentOrder, error)
	GetPaymentOrder(ctx context.Context, id uint64) (*PaymentOrder, error)
	GetPaymentOrderByMerchantOrder(ctx context.Context, req GetPaymentOrderByMerchantOrderRequest) (*PaymentOrder, error)
	QueryPaymentOrder(ctx context.Context, id uint64) (*PaymentOrder, error)
	CheckPaymentOrderByPaymentNo(ctx context.Context, req CheckPaymentOrderByPaymentNoRequest) (*PaymentOrderCheckResult, error)
	CancelPaymentOrder(ctx context.Context, id uint64) (*PaymentOrder, error)
	ListPaymentOrders(ctx context.Context, req ListPaymentOrdersRequest) (*ListPaymentOrdersResponse, error)
	ListSupportedCurrencies(ctx context.Context) (*ListSupportedCurrenciesResponse, error)
	ListAvailablePaymentChannels(ctx context.Context, req ListAvailablePaymentChannelsRequest) (*ListAvailablePaymentChannelsResponse, error)
	Close() error
}

type CreatePaymentOrderRequest struct {
	MerchantCode    string `json:"merchantCode"`
	MerchantOrderID string `json:"merchantOrderId"`
	// TenantID is the workspace owner/billing boundary for the order.
	TenantID string `json:"tenantId"`
	// PayerUserID is optional payer/actor metadata and must not be used as owner scope.
	PayerUserID      string  `json:"payerUserId,omitempty"`
	OrderAmount      float64 `json:"orderAmount"`
	OrderCurrency    string  `json:"orderCurrency"`
	PaymentCurrency  string  `json:"paymentCurrency"`
	ChannelCode      string  `json:"channelCode"`
	IdempotencyKey   string  `json:"idempotencyKey,omitempty"`
	SuccessReturnURL string  `json:"successReturnUrl,omitempty"`
	Metadata         JSONMap `json:"metadata,omitempty"`
}

type GetPaymentOrderByMerchantOrderRequest struct {
	MerchantCode    string `json:"merchantCode"`
	MerchantOrderID string `json:"merchantOrderId"`
}

type CheckPaymentOrderByPaymentNoRequest struct {
	PaymentNo       string `json:"paymentNo"`
	MerchantCode    string `json:"merchantCode"`
	MerchantOrderID string `json:"merchantOrderId"`
}

type PaymentOrderCheckResult struct {
	Order          *PaymentOrder `json:"order"`
	Result         string        `json:"result"`
	NextEligibleAt string        `json:"nextEligibleAt,omitempty"`
}

type ListPaymentOrdersRequest struct {
	Page         int
	PageSize     int
	MerchantCode string
	// TenantID filters the workspace owner/billing boundary.
	TenantID string
	// PayerUserID filters optional payer/actor metadata, not owner scope.
	PayerUserID string
	Status      string
}

type ListPaymentOrdersResponse struct {
	Items     []PaymentOrder `json:"items"`
	Total     int64          `json:"total"`
	TotalPage int64          `json:"totalPage"`
	Page      int            `json:"page"`
	PageSize  int            `json:"pageSize"`
}

type ListSupportedCurrenciesResponse struct {
	Currencies []string `json:"currencies"`
}

type ListAvailablePaymentChannelsRequest struct {
	MerchantCode    string  `json:"merchantCode"`
	OrderAmount     float64 `json:"orderAmount,omitempty"`
	OrderCurrency   string  `json:"orderCurrency"`
	PaymentCurrency string  `json:"paymentCurrency"`
}

type ListAvailablePaymentChannelsResponse struct {
	Items []AvailablePaymentChannel `json:"items"`
}

type AvailablePaymentChannel struct {
	ChannelCode     string  `json:"channelCode"`
	Name            string  `json:"name"`
	Category        string  `json:"category"`
	CategoryName    string  `json:"categoryName"`
	CategoryOrder   int     `json:"categoryOrder"`
	MethodOrder     int     `json:"methodOrder"`
	PaymentCurrency string  `json:"paymentCurrency"`
	PaymentAmount   float64 `json:"paymentAmount"`
}

type PaymentOrder struct {
	ID              uint64 `json:"id"`
	PaymentNo       string `json:"paymentNo"`
	MerchantCode    string `json:"merchantCode"`
	MerchantOrderID string `json:"merchantOrderId"`
	// TenantID is the workspace owner/billing boundary for the order.
	TenantID string `json:"tenantId"`
	// PayerUserID is optional payer/actor metadata and must not be used as owner scope.
	PayerUserID         string  `json:"payerUserId"`
	OrderAmount         float64 `json:"orderAmount"`
	OrderCurrency       string  `json:"orderCurrency"`
	SettlementAmount    float64 `json:"settlementAmount"`
	SettlementCurrency  string  `json:"settlementCurrency"`
	PaymentAmount       float64 `json:"paymentAmount"`
	PaymentCurrency     string  `json:"paymentCurrency"`
	OrderExchangeRate   float64 `json:"orderExchangeRate"`
	PaymentExchangeRate float64 `json:"paymentExchangeRate"`
	Provider            string  `json:"provider"`
	ChannelCode         string  `json:"channelCode"`
	ChannelID           uint64  `json:"channelId"`
	MerchantAccountID   uint64  `json:"merchantAccountId"`
	ProviderPaymentID   string  `json:"providerPaymentId"`
	CheckoutURL         string  `json:"checkoutUrl"`
	Status              string  `json:"status"`
	FailureReason       string  `json:"failureReason"`
	Metadata            JSONMap `json:"metadata"`
	ExpiresAt           string  `json:"expiresAt"`
	CreatedAt           string  `json:"createdAt"`
	UpdatedAt           string  `json:"updatedAt"`
}

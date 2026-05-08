package xpaymentsdk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	pb "xpayment-svc/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreatePaymentOrderGRPCSuccessDoesNotCallHTTP(t *testing.T) {
	var httpCalls atomic.Int32
	fakeGRPC := &fakePaymentServiceClient{createResp: protoOrderFixture()}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpCalls.Add(1)
		writeEnvelope(t, w, http.StatusOK, orderFixture())
	}))
	defer server.Close()

	c := &client{
		grpc:    fakeGRPC,
		http:    newHTTPClient(server.URL, server.Client()),
		timeout: time.Second,
	}
	order, err := c.CreatePaymentOrder(context.Background(), createFixture())
	if err != nil {
		t.Fatalf("CreatePaymentOrder returned error: %v", err)
	}
	if order.PaymentNo != "P1" {
		t.Fatalf("expected grpc payment no P1, got %q", order.PaymentNo)
	}
	if httpCalls.Load() != 0 {
		t.Fatalf("expected no http fallback calls, got %d", httpCalls.Load())
	}
	if got := fakeGRPC.lastCreateReq.GetSuccessReturnUrl(); got != "https://console.example.com/wallet" {
		t.Fatalf("expected success return url to be sent over grpc, got %q", got)
	}
	if got := fakeGRPC.lastCreateReq.GetCallbackBaseUrl(); got != "http://app:6060/internal/payment/xpayment/callback" {
		t.Fatalf("expected business callback url to be sent over grpc, got %q", got)
	}
}

func TestCreatePaymentOrderFallbacksOnUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/payment/orders" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		var req CreatePaymentOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.CallbackBaseURL != "http://app:6060/internal/payment/xpayment/callback" {
			t.Fatalf("expected business callback url over http fallback, got %q", req.CallbackBaseURL)
		}
		writeEnvelope(t, w, http.StatusOK, orderFixture())
	}))
	defer server.Close()

	c := &client{
		grpc:    &fakePaymentServiceClient{createErr: status.Error(codes.Unavailable, "transport unavailable")},
		http:    newHTTPClient(server.URL, server.Client()),
		timeout: time.Second,
	}
	order, err := c.CreatePaymentOrder(context.Background(), createFixture())
	if err != nil {
		t.Fatalf("CreatePaymentOrder returned error: %v", err)
	}
	if order.PaymentNo != "P1" {
		t.Fatalf("expected fallback payment no P1, got %q", order.PaymentNo)
	}
}

func TestCreatePaymentOrderBusinessErrorDoesNotFallback(t *testing.T) {
	var httpCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpCalls.Add(1)
		writeEnvelope(t, w, http.StatusOK, orderFixture())
	}))
	defer server.Close()

	c := &client{
		grpc:    &fakePaymentServiceClient{createErr: status.Error(codes.InvalidArgument, "invalid payment currency")},
		http:    newHTTPClient(server.URL, server.Client()),
		timeout: time.Second,
	}
	if _, err := c.CreatePaymentOrder(context.Background(), createFixture()); err == nil {
		t.Fatal("expected business error")
	}
	if httpCalls.Load() != 0 {
		t.Fatalf("expected no http fallback calls, got %d", httpCalls.Load())
	}
}

func TestPureHTTPModeListSupportedCurrencies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/payment/currencies" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		writeEnvelope(t, w, http.StatusOK, ListSupportedCurrenciesResponse{Currencies: []string{"USD", "CNY"}})
	}))
	defer server.Close()

	raw, err := NewClient(Config{HTTPBaseURL: server.URL, PreferHTTP: true, Timeout: time.Second})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	resp, err := raw.ListSupportedCurrencies(context.Background())
	if err != nil {
		t.Fatalf("ListSupportedCurrencies returned error: %v", err)
	}
	if len(resp.Currencies) != 2 || resp.Currencies[1] != "CNY" {
		t.Fatalf("unexpected currencies: %#v", resp.Currencies)
	}
}

func TestHTTPEnvelopeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEnvelopeWithCode(t, w, http.StatusBadRequest, "bad request")
	}))
	defer server.Close()

	c := newHTTPClient(server.URL, server.Client())
	_, err := c.listSupportedCurrencies(context.Background())
	if err == nil {
		t.Fatal("expected http envelope error")
	}
	httpErr, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.Code != http.StatusBadRequest {
		t.Fatalf("expected code 400, got %d", httpErr.Code)
	}
}

func createFixture() CreatePaymentOrderRequest {
	return CreatePaymentOrderRequest{
		MerchantCode: "xai-wallet", MerchantOrderID: "m1", UserID: "550e8400-e29b-41d4-a716-446655440000",
		OrderAmount: 10, OrderCurrency: "USD", PaymentCurrency: "CNY", ChannelCode: "alipay", IdempotencyKey: "idem-1",
		CallbackBaseURL:  "http://app:6060/internal/payment/xpayment/callback",
		SuccessReturnURL: "https://console.example.com/wallet",
		Metadata:         JSONMap{"source": "test"},
	}
}

func orderFixture() PaymentOrder {
	return PaymentOrder{
		ID: 1, PaymentNo: "P1", MerchantCode: "xai-wallet", MerchantOrderID: "m1", UserID: "550e8400-e29b-41d4-a716-446655440000",
		OrderAmount: 10, OrderCurrency: "USD", SettlementAmount: 10, SettlementCurrency: "USD", PaymentAmount: 72,
		PaymentCurrency: "CNY", OrderExchangeRate: 1, PaymentExchangeRate: 7.2, Provider: "mock", ChannelCode: "alipay",
		ChannelID: 2, MerchantAccountID: 3, ProviderPaymentID: "provider-1", CheckoutURL: "https://pay.test/checkout",
		Status: "checkout_created", Metadata: JSONMap{"source": "test"},
	}
}

func protoOrderFixture() *pb.PaymentOrderResponse {
	return &pb.PaymentOrderResponse{
		Id: 1, PaymentNo: "P1", MerchantCode: "xai-wallet", MerchantOrderId: "m1", UserId: "550e8400-e29b-41d4-a716-446655440000",
		OrderAmount: 10, OrderCurrency: "USD", SettlementAmount: 10, SettlementCurrency: "USD", PaymentAmount: 72,
		PaymentCurrency: "CNY", OrderExchangeRate: 1, PaymentExchangeRate: 7.2, Provider: "mock", ChannelCode: "alipay",
		ChannelId: 2, MerchantAccountId: 3, ProviderPaymentId: "provider-1", CheckoutUrl: "https://pay.test/checkout",
		Status: "checkout_created", MetadataJson: `{"source":"test"}`,
	}
}

func writeEnvelope(t *testing.T, w http.ResponseWriter, statusCode int, data any) {
	t.Helper()
	payload, err := json.Marshal(map[string]any{"code": statusCode, "message": "success", "data": data})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(payload)
}

func writeEnvelopeWithCode(t *testing.T, w http.ResponseWriter, code int, message string) {
	t.Helper()
	payload, err := json.Marshal(map[string]any{"code": code, "message": message})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(payload)
}

type fakePaymentServiceClient struct {
	createResp    *pb.PaymentOrderResponse
	createErr     error
	lastCreateReq *pb.CreatePaymentOrderRequest
}

func (f *fakePaymentServiceClient) CreatePaymentOrder(_ context.Context, req *pb.CreatePaymentOrderRequest, _ ...grpc.CallOption) (*pb.PaymentOrderResponse, error) {
	f.lastCreateReq = req
	return f.createResp, f.createErr
}

func (f *fakePaymentServiceClient) GetPaymentOrder(context.Context, *pb.GetPaymentOrderRequest, ...grpc.CallOption) (*pb.PaymentOrderResponse, error) {
	return protoOrderFixture(), nil
}

func (f *fakePaymentServiceClient) GetPaymentOrderByMerchantOrder(context.Context, *pb.GetPaymentOrderByMerchantOrderRequest, ...grpc.CallOption) (*pb.PaymentOrderResponse, error) {
	return protoOrderFixture(), nil
}

func (f *fakePaymentServiceClient) QueryPaymentOrder(context.Context, *pb.GetPaymentOrderRequest, ...grpc.CallOption) (*pb.PaymentOrderResponse, error) {
	return protoOrderFixture(), nil
}

func (f *fakePaymentServiceClient) CancelPaymentOrder(context.Context, *pb.GetPaymentOrderRequest, ...grpc.CallOption) (*pb.PaymentOrderResponse, error) {
	return protoOrderFixture(), nil
}

func (f *fakePaymentServiceClient) ListPaymentOrders(context.Context, *pb.ListPaymentOrdersRequest, ...grpc.CallOption) (*pb.ListPaymentOrdersResponse, error) {
	return &pb.ListPaymentOrdersResponse{Items: []*pb.PaymentOrderResponse{protoOrderFixture()}, Total: 1, Page: 1, PageSize: 20}, nil
}

func (f *fakePaymentServiceClient) ListSupportedCurrencies(context.Context, *pb.ListSupportedCurrenciesRequest, ...grpc.CallOption) (*pb.ListSupportedCurrenciesResponse, error) {
	return &pb.ListSupportedCurrenciesResponse{Currencies: []string{"USD", "CNY"}}, nil
}

func (f *fakePaymentServiceClient) ListAvailablePaymentChannels(context.Context, *pb.ListAvailablePaymentChannelsRequest, ...grpc.CallOption) (*pb.ListAvailablePaymentChannelsResponse, error) {
	return &pb.ListAvailablePaymentChannelsResponse{Items: []*pb.AvailablePaymentChannel{{ChannelCode: "alipay", Provider: "mock"}}}, nil
}

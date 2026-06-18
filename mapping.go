package xpaymentsdk

import (
	"fmt"

	"github.com/bytedance/sonic"
	pb "xpayment-svc/proto"
)

func createToProto(req CreatePaymentOrderRequest) (*pb.CreatePaymentOrderRequest, error) {
	metadata, err := marshalMetadata(req.Metadata)
	if err != nil {
		return nil, err
	}
	return &pb.CreatePaymentOrderRequest{
		MerchantCode:     req.MerchantCode,
		MerchantOrderId:  req.MerchantOrderID,
		TenantId:         req.TenantID,
		UserId:           req.UserID,
		OrderAmount:      req.OrderAmount,
		OrderCurrency:    req.OrderCurrency,
		PaymentCurrency:  req.PaymentCurrency,
		ChannelCode:      req.ChannelCode,
		IdempotencyKey:   req.IdempotencyKey,
		CallbackBaseUrl:  req.CallbackBaseURL,
		SuccessReturnUrl: req.SuccessReturnURL,
		MetadataJson:     metadata,
	}, nil
}

func orderFromProto(order *pb.PaymentOrderResponse) (*PaymentOrder, error) {
	if order == nil {
		return nil, nil
	}
	metadata, err := unmarshalMetadata(order.GetMetadataJson())
	if err != nil {
		return nil, err
	}
	return &PaymentOrder{
		ID:                  order.GetId(),
		PaymentNo:           order.GetPaymentNo(),
		MerchantCode:        order.GetMerchantCode(),
		MerchantOrderID:     order.GetMerchantOrderId(),
		TenantID:            order.GetTenantId(),
		UserID:              order.GetUserId(),
		OrderAmount:         order.GetOrderAmount(),
		OrderCurrency:       order.GetOrderCurrency(),
		SettlementAmount:    order.GetSettlementAmount(),
		SettlementCurrency:  order.GetSettlementCurrency(),
		PaymentAmount:       order.GetPaymentAmount(),
		PaymentCurrency:     order.GetPaymentCurrency(),
		OrderExchangeRate:   order.GetOrderExchangeRate(),
		PaymentExchangeRate: order.GetPaymentExchangeRate(),
		Provider:            order.GetProvider(),
		ChannelCode:         order.GetChannelCode(),
		ChannelID:           order.GetChannelId(),
		MerchantAccountID:   order.GetMerchantAccountId(),
		ProviderPaymentID:   order.GetProviderPaymentId(),
		CheckoutURL:         order.GetCheckoutUrl(),
		Status:              order.GetStatus(),
		FailureReason:       order.GetFailureReason(),
		Metadata:            metadata,
		ExpiresAt:           order.GetExpiresAt(),
		CreatedAt:           order.GetCreatedAt(),
		UpdatedAt:           order.GetUpdatedAt(),
	}, nil
}

func listOrdersFromProto(resp *pb.ListPaymentOrdersResponse) (*ListPaymentOrdersResponse, error) {
	if resp == nil {
		return nil, nil
	}
	items := make([]PaymentOrder, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		order, err := orderFromProto(item)
		if err != nil {
			return nil, err
		}
		if order != nil {
			items = append(items, *order)
		}
	}
	return &ListPaymentOrdersResponse{Items: items, Total: resp.GetTotal(), Page: int(resp.GetPage()), PageSize: int(resp.GetPageSize())}, nil
}

func availableChannelsFromProto(resp *pb.ListAvailablePaymentChannelsResponse) *ListAvailablePaymentChannelsResponse {
	if resp == nil {
		return nil
	}
	items := make([]AvailablePaymentChannel, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		if item == nil {
			continue
		}
		items = append(items, AvailablePaymentChannel{
			ChannelCode:       item.GetChannelCode(),
			Provider:          item.GetProvider(),
			ChannelID:         item.GetChannelId(),
			MerchantAccountID: item.GetMerchantAccountId(),
			Name:              item.GetName(),
			PaymentCurrency:   item.GetPaymentCurrency(),
			PaymentAmount:     item.GetPaymentAmount(),
		})
	}
	return &ListAvailablePaymentChannelsResponse{Items: items}
}

func marshalMetadata(metadata JSONMap) (string, error) {
	if len(metadata) == 0 {
		return "", nil
	}
	value, err := sonic.MarshalString(metadata)
	if err != nil {
		return "", fmt.Errorf("marshal payment metadata: %w", err)
	}
	return value, nil
}

func unmarshalMetadata(value string) (JSONMap, error) {
	if value == "" {
		return JSONMap{}, nil
	}
	metadata := JSONMap{}
	if err := sonic.UnmarshalString(value, &metadata); err != nil {
		return nil, fmt.Errorf("unmarshal payment metadata: %w", err)
	}
	return metadata, nil
}

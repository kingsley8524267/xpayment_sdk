package xpaymentsdk_test

import (
	"google.golang.org/protobuf/encoding/protojson"
	"strings"
	"testing"
	pb "xpayment-sdk/proto"
)

func TestPaymentAccountJSONContract(t *testing.T) {
	var request pb.CreatePaymentOrderRequest
	if err := protojson.Unmarshal([]byte(`{"tenantId":"tenant","payerAccountId":"account"}`), &request); err != nil || request.TenantId != "tenant" || request.PayerAccountId != "account" {
		t.Fatal("canonical ownership/identity", err)
	}
	if err := protojson.Unmarshal([]byte(`{"tenantId":"tenant","payerUserId":"old"}`), &request); err == nil {
		t.Fatal("retired payer field accepted")
	}
	body, err := protojson.Marshal(&pb.PaymentOrderResponse{TenantId: "tenant", PayerAccountId: "account"})
	if err != nil || strings.Contains(string(body), "payerUserId") || !strings.Contains(string(body), "payerAccountId") {
		t.Fatal("noncanonical payment response", err)
	}
}

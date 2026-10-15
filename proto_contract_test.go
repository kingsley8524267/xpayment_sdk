package xpaymentsdk_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	paymentv1 "xpayment-sdk/proto"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
)

const paymentWireDescriptorSHA256 = "b296e33e7d3455ccc55c756e009e57cdff1f9ca865c29687c4f0ebbf29007a77"

func TestSDKOwnsPaymentProtoContract(t *testing.T) {
	service := paymentv1.File_proto_payment_proto.Services().ByName("PaymentService")
	if service == nil {
		t.Fatal("payment.v1.PaymentService is missing")
	}
	if got := string(service.FullName()); got != "payment.v1.PaymentService" {
		t.Fatalf("unexpected protobuf service name: %s", got)
	}

	goMod, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	if strings.Contains(string(goMod), "xpayment-svc") {
		t.Fatal("xpayment SDK must not depend on xpayment service")
	}
}

func TestPaymentWireContractDescriptor(t *testing.T) {
	descriptor := protodesc.ToFileDescriptorProto(paymentv1.File_proto_payment_proto)
	descriptor.SourceCodeInfo = nil
	if descriptor.Options != nil {
		descriptor.Options.GoPackage = nil
	}
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(descriptor)
	if err != nil {
		t.Fatalf("marshal payment wire descriptor: %v", err)
	}
	sum := sha256.Sum256(encoded)
	if got := hex.EncodeToString(sum[:]); got != paymentWireDescriptorSHA256 {
		t.Fatalf("payment wire descriptor changed: got %s want %s", got, paymentWireDescriptorSHA256)
	}
}

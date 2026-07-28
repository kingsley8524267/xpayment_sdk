PROTO_FILES := proto/payment.proto
PROTO_OUTPUTS := proto/payment.pb.go proto/payment_grpc.pb.go
PROTO_CMD := protoc --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. $(PROTO_FILES)
PROTOC_VERSION := libprotoc 33.2
PROTOC_GEN_GO_VERSION := protoc-gen-go v1.36.11
PROTOC_GEN_GO_GRPC_VERSION := protoc-gen-go-grpc 1.6.0

.PHONY: proto proto-check proto-toolchain-check

proto-toolchain-check:
	@test "$$(protoc --version)" = "$(PROTOC_VERSION)" || { echo "protoc must be $(PROTOC_VERSION)" >&2; exit 1; }
	@test "$$(protoc-gen-go --version)" = "$(PROTOC_GEN_GO_VERSION)" || { echo "protoc-gen-go must be $(PROTOC_GEN_GO_VERSION)" >&2; exit 1; }
	@test "$$(protoc-gen-go-grpc --version)" = "$(PROTOC_GEN_GO_GRPC_VERSION)" || { echo "protoc-gen-go-grpc must be $(PROTOC_GEN_GO_GRPC_VERSION)" >&2; exit 1; }

proto: proto-toolchain-check
	$(PROTO_CMD)

proto-check: proto
	@git diff --exit-code -- $(PROTO_FILES) $(PROTO_OUTPUTS)

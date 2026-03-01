PROTO_DIR = proto
PB_OUT_DIR = pb

.PHONY: proto-tg-service


ensure-pb-dirs:
	@mkdir -p $(PB_OUT_DIR)


proto-tg-service: ensure-pb-dirs
	protoc -I $(PROTO_DIR) \
	--go_out=$(PB_OUT_DIR) --go_opt=paths=source_relative \
	--go-grpc_out=$(PB_OUT_DIR) --go-grpc_opt=paths=source_relative \
	$(PROTO_DIR)/v1/*.proto
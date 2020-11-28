
.PHONY: test protobuf

protobuf:
	cd ./pkg/inet256grpc && ./build.sh
test:
	go test ./pkg/...

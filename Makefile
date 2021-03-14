
.PHONY: test protobuf drop-replace add-replace

protobuf:
	cd ./pkg/inet256grpc && ./build.sh
	cd ./networks/kadsrnet && ./build.sh

install:
	go install ./cmd/inet256	

test: protobuf
	go test --race ./pkg/...
	go test --race ./client/go_client/...
	go test --race ./networks/...

testv: protobuf
	go test --race -v -count=1 ./pkg/...
	go test --race -v -count=1 ./client/go_client/...
	go test --race -v -count=1 ./networks/...

drop-replace:
	go mod edit -dropreplace github.com/brendoncarroll/go-p2p

add-replace:
	go mod edit -replace github.com/brendoncarroll/go-p2p=../../brendoncarroll/go-p2p


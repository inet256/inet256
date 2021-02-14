
.PHONY: test protobuf drop-replace add-replace

protobuf:
	cd ./pkg/inet256grpc && ./build.sh
	cd ./pkg/kadsrnet && ./build.sh

install:
	go install ./cmd/inet256	

test:
	go test --race ./pkg/...
	go test --race ./client/go_client/...

testv: protobuf
	go test --race -v -count=1 ./pkg/...
	go test --race -v -count=1 ./client/go_client/...

drop-replace:
	go mod edit -dropreplace github.com/brendoncarroll/go-p2p

add-replace:
	go mod edit -replace github.com/brendoncarroll/go-p2p=../../brendoncarroll/go-p2p


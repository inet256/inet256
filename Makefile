
.PHONY: test protobuf drop-replace add-replace

protobuf:
	cd ./pkg/inet256grpc && ./build.sh
test:
	go test --race ./pkg/...

testv:
	go test --race -v -count=1 ./pkg/...

drop-replace:
	go mod edit -dropreplace github.com/brendoncarroll/go-p2p

add-replace:
	go mod edit -replace github.com/brendoncarroll/go-p2p=../../brendoncarroll/go-p2p


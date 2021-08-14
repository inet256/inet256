
.PHONY: test protobuf drop-replace add-replace

protobuf:
	cd ./pkg/inet256grpc && ./build.sh

install:
	go install ./cmd/inet256

build:
	GOOS=darwin GOARCH=amd64 go build -o ./build-outputs/inet256_darwin-amd64 ./cmd/inet256 
	GOOS=linux GOARCH=amd64 go build -o ./build-outputs/inet256_linux-amd64 ./cmd/inet256 
	GOOS=windows GOARCH=amd64 go build -o ./build-outputs/inet256_windows-amd64 ./cmd/inet256

test: protobuf
	go test --race ./pkg/...
	go test --race ./client/go_client/...
	go test --race ./networks/...
	go test --race ./cmd/...

testv: protobuf
	go test --race -v -count=1 ./pkg/...
	go test --race -v -count=1 ./client/go_client/...
	go test --race -v -count=1 ./networks/...
	go test --race -v -count=1 ./cmd/...

drop-replace:
	go mod edit -dropreplace github.com/brendoncarroll/go-p2p

add-replace:
	go mod edit -replace github.com/brendoncarroll/go-p2p=../../brendoncarroll/go-p2p


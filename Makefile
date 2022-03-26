
.PHONY: test protobuf drop-replace add-replace build

clean:
	-rm -r ./build/*

protobuf:
	cd ./pkg/inet256grpc && ./build.sh

install:
	go install ./cmd/inet256

build: protobuf
	GOOS=darwin GOARCH=amd64 ./etc/build_go_binary.sh build/inet256_darwin-amd64_$(TAG) ./cmd/inet256
	GOOS=linux GOARCH=amd64 ./etc/build_go_binary.sh build/inet256_linux-amd64_$(TAG) ./cmd/inet256
	GOOS=windows GOARCH=amd64 ./etc/build_go_binary.sh build/inet256_windows-amd64_$(TAG) ./cmd/inet256

test: protobuf
	go test --race ./pkg/...
	go test --race ./client/go_client/...
	go test --race ./networks/...
	go test --race ./cmd/...
	go test --race ./e2etest

testv: protobuf
	go test --race -v -count=1 ./pkg/...
	go test --race -v -count=1 ./client/go_client/...
	go test --race -v -count=1 ./networks/...
	go test --race -v -count=1 ./cmd/...
	go test --race -v -count=1 ./e2etest

docker:
	docker build -t inet256:local .

drop-replace:
	go mod edit -dropreplace github.com/brendoncarroll/go-p2p

add-replace:
	go mod edit -replace github.com/brendoncarroll/go-p2p=../../brendoncarroll/go-p2p


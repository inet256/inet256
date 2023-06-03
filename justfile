
clean:
	-rm -r ./out/*

protobuf:
	cd ./pkg/discovery/centraldisco/internal && ./build.sh

# install-unix builds a binary and copies it to /usr/local/bin
install-unix: build
	cp ./out/inet256 /usr/local/bin/inet256

install-systemd:
	cp etc/systemd/* /etc/systemd/system/
	systemctl daemon-reload

# builds inet256 for the local platform
build:
	./etc/build_go_binary.sh ./out/inet256 ./cmd/inet256

# build-release builds all binaries for a release
build-release: protobuf
	GOOS=darwin GOARCH=amd64 ./etc/build_go_binary.sh out/inet256_darwin_amd64_$(TAG) ./cmd/inet256
	GOOS=darwin GOARCH=arm64 ./etc/build_go_binary.sh out/inet256_darwin_arm64_$(TAG) ./cmd/inet256
	GOOS=linux  GOARCH=amd64 ./etc/build_go_binary.sh out/inet256_linux_amd64_$(TAG) ./cmd/inet256
	GOOS=linux  GOARCH=arm64 ./etc/build_go_binary.sh out/inet256_linux_arm64_$(TAG) ./cmd/inet256

test: protobuf
	go test --race ./internal/...
	go test --race ./pkg/...
	go test --race ./client/go/...
	go test --race ./networks/...
	go test --race ./cmd/...
	go test ./e2etest

testv: protobuf
	go test --race -v ./internal/...
	go test --race -v -count=1 ./pkg/...
	go test --race -v -count=1 ./client/go/...
	go test --race -v -count=1 ./networks/...
	go test --race -v -count=1 ./cmd/...
	go test --race -v -count=1 ./e2etest

bench:
	go test -bench=. -run Benchmark ./...

docker:
	docker build -t inet256:local .

drop-replace:
	go mod edit -dropreplace github.com/brendoncarroll/go-p2p

add-replace:
	go mod edit -replace github.com/brendoncarroll/go-p2p=../../brendoncarroll/go-p2p


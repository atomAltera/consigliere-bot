TAG=registry.nuclight.org/consigliere-tg-bot:latest

.PHONY: cleanup
cleanup:
	rm -rf ./bin

.PHONY: lint
lint:
	go tool golangci-lint run

.PHONY: test
test:
	go test ./...

.PHONY: test-verbose
test-verbose:
	go test -v ./...

.PHONY: run
run:
	go run ./cmd/consigliere

.PHONY: build
build: cleanup
	mkdir -p ./bin
	GOOS=darwin GOARCH=arm64 go build -ldflags "\
		-X main.Version=$$(git rev-parse --short HEAD) \
		-X main.BuildDate=$$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
		-o ./bin/consigliere_darwin_arm64 ./cmd/consigliere
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "\
		-X main.Version=$$(git rev-parse --short HEAD) \
		-X main.BuildDate=$$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
		-o ./bin/consigliere_linux_amd64 ./cmd/consigliere

.PHONY: docker_build
docker_build: build
	docker build -t $(TAG) --platform linux/amd64 .

.PHONY: docker_push
docker_push: docker_build
	docker push $(TAG)

.PHONY: pull_db
pull_db:
	scp nuclight.org:consigliere-tg-bot/db/consigliere.sqlite ./db/consigliere.sqlite

.PHONY: push_db
push_db:
	scp ./db/consigliere.sqlite nuclight.org:consigliere-tg-bot/db/consigliere.sqlite

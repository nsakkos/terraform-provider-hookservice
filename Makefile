BINARY=terraform-provider-hookservice
VERSION?=0.1.0
OS_ARCH?=linux_amd64
INSTALL_DIR=~/.terraform.d/plugins/registry.terraform.io/canonical/hookservice/$(VERSION)/$(OS_ARCH)

.PHONY: build test testacc lint fmt vet install clean

build:
	go build -o $(BINARY)

test:
	go test ./... -count=1

testacc:
	TF_ACC=1 go test ./... -v -count=1 -timeout 120s

lint: fmt vet

fmt:
	@echo "==> Checking formatting..."
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:"; gofmt -l .; exit 1)

vet:
	go vet ./...

install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY) $(INSTALL_DIR)/

clean:
	rm -f $(BINARY)

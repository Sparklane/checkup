.PHONY: build

all: build

build-docker:
	@docker run --net=host --rm \
		-v $(PWD):/project \
		-w /project golang:1.13.0 bash -c "cd cmd/checkup; go get -v -d; CGO_ENABLED=0 go build -v -ldflags '-s' -o ../../checkup"

build:
	@cd cmd/checkup; go get -v -d; CGO_ENABLED=0 go build -v -ldflags '-s' -o ../../checkup

test:
	@./checkup --config=config-test.json --v

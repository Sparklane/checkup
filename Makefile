.PHONY: build

all: build

build:
	@docker run --net=host --rm \
		-v $(PWD):/project \
		-w /project golang:1.11.5 bash -c "cd cmd/checkup; go get -v -d; go build -v -ldflags '-s' -o ../../checkup"

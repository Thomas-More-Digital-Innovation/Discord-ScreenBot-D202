.PHONY: build test run clean docker-build

BINARY=screenbot

build:
	go build -o $(BINARY) .

test:
	go test -v ./...

run: build
	./$(BINARY) run --config config.yaml

screens: build
	./$(BINARY) screens --config config.example.yaml

clean:
	rm -f $(BINARY)

docker-build:
	docker build -t discord-screenbot .

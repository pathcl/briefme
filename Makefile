BIN     := briefme
CMD     := ./cmd/briefme
CONFIG  := config.yaml

.PHONY: build run serve test clean

build:
	go build -o $(BIN) $(CMD)

run: build
	./$(BIN) -config $(CONFIG)

serve: build
	./$(BIN) serve -config $(CONFIG)

test:
	go test ./...

clean:
	rm -f $(BIN)

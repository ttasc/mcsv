all: build

# Build the binary
build:
	@echo "Building..."
	@go build -o bin/craftt ./src
	@echo "Done! Binary now is in bin/craftt"

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -f bin/craftt

.PHONY: all build clean

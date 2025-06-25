all: build

build:
	@echo "Building..."

	@go build -o bin/server server/*

	@echo "Done!"

# Clean the binary
# clean:
# 	@echo "Cleaning..."
# 	@rm -f bin/main

.PHONY: all build

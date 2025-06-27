all: build scp

build:
	@echo "Building..."

	@go build -o bin/server ./server

scp:
	@echo "Copying to server (tailscale)..."

	@scp -i ~/.ssh/home bin/server mcsv@100.101.0.1:~/server

# Clean the binary
# clean:
# 	@echo "Cleaning..."
# 	@rm -f bin/main

.PHONY: all build scp

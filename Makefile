all: build scp

build:
	@echo "Building..."

	@go build -o bin/gova ./server

scp:
	@echo "Copying to server (tailscale)..."

	@scp -i ~/.ssh/home bin/gova mcsv@100.101.0.1:~/gova

# Clean the binary
# clean:
# 	@echo "Cleaning..."
# 	@rm -f bin/main

.PHONY: all build scp

all: goja scp

goja:
	@echo "Building..."

	@go build -o bin/goja ./goja

scp:
	@echo "Copying to server (tailscale)..."

	@scp -i ~/.ssh/home bin/goja mcsv@100.101.0.1:~/goja

# Clean the binary
# clean:
# 	@echo "Cleaning..."
# 	@rm -f bin/main

.PHONY: all goja scp

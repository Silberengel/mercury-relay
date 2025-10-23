#!/bin/bash

# Simple test for authentication and note publishing
export NSEC="6fadb1c15f77cbb4074c4b0facd15654c81e79e295a7039af19c29b1e2d7e3b9"

echo "Testing simple authentication and note publishing..."
# Use printf to provide all inputs at once
printf "2\ny\n\n6\n1\nHello from Mercury Relay TUI!\nnostr,mercury,relay\n2700e0802d1d0b9ad24ba2599b95d03a57a94016db37f83718a8038977d92aac\nn\nn\nn\nn\nn\nn\nn\nn\nn\nn\n" | go run cmd/mercury-admin/main.go --tui --config config.local.yaml

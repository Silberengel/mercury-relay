#!/bin/bash

# Test script for TUI with NIP-42 authentication
export NSEC="6fadb1c15f77cbb4074c4b0facd15654c81e79e295a7039af19c29b1e2d7e3b9"

echo "Testing TUI with NIP-42 authentication..."
# Create a comprehensive input file with all necessary inputs
cat > test_input.txt << EOF
2
y

6
1
Hello from Mercury Relay TUI!
nostr,mercury,relay
2700e0802d1d0b9ad24ba2599b95d03a57a94016db37f83718a8038977d92aac
n
n
n
n
n
n
n
n
n
n
EOF

# Run the TUI with the input file
go run cmd/mercury-admin/main.go --tui --config config.local.yaml < test_input.txt

# Clean up
rm test_input.txt

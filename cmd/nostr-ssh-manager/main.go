package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"mercury-relay/internal/config"

	"github.com/nbd-wtf/go-nostr"
)

func main() {
	fmt.Println("🔐 Mercury Relay Nostr SSH Key Manager")
	fmt.Println("=====================================")

	// Load configuration
	var configPath = flag.String("config", "../../config.yaml", "Path to configuration file")
	var relayURL = flag.String("relay", "http://localhost:8082", "Relay URL for authentication")
	flag.Parse()

	_, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Check for Nostr private key
	privateKey := os.Getenv("MERCURY_PRIVATE_KEY")
	if privateKey == "" {
		fmt.Println("❌ Error: MERCURY_PRIVATE_KEY environment variable not set")
		fmt.Println("Please set your Nostr private key:")
		fmt.Println("  export MERCURY_PRIVATE_KEY=\"nsec1your-private-key\"")
		os.Exit(1)
	}

	// Authenticate with Nostr
	npub, err := authenticateWithNostr(*relayURL, privateKey)
	if err != nil {
		log.Fatalf("Failed to authenticate with Nostr: %v", err)
	}

	fmt.Printf("✅ Authenticated as: %s\n", npub)
	fmt.Println("SSH Key Manager - Type 'help' for commands")
	fmt.Println()

	// Start interactive terminal
	runInteractiveTerminal(*relayURL, npub)
}

func authenticateWithNostr(relayURL, privateKey string) (string, error) {
	fmt.Println("🔑 Authenticating with Nostr...")

	// Get public key from private key
	pubkey, err := nostr.GetPublicKey(privateKey)
	if err != nil {
		return "", fmt.Errorf("invalid private key: %w", err)
	}

	// Get challenge
	challenge, err := getChallenge(relayURL)
	if err != nil {
		return "", fmt.Errorf("failed to get challenge: %w", err)
	}

	// Create and sign auth event
	authEvent := &nostr.Event{
		Kind:      22242,
		CreatedAt: nostr.Now(),
		Tags: nostr.Tags{
			{"relay", relayURL},
			{"challenge", challenge},
		},
		Content: "",
		PubKey:  pubkey,
	}

	// Sign the event
	if err := authEvent.Sign(privateKey); err != nil {
		return "", fmt.Errorf("failed to sign auth event: %w", err)
	}

	// Submit authentication
	if err := submitAuth(relayURL, authEvent); err != nil {
		return "", fmt.Errorf("failed to submit auth: %w", err)
	}

	return pubkey, nil
}

func getChallenge(relayURL string) (string, error) {
	resp, err := http.Get(relayURL + "/api/v1/nostr/challenge")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get challenge: status %d", resp.StatusCode)
	}

	var result struct {
		Challenge string `json:"challenge"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Challenge, nil
}

func submitAuth(relayURL string, event *nostr.Event) error {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return err
	}

	reqBody := map[string]interface{}{
		"event": json.RawMessage(eventJSON),
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	resp, err := http.Post(relayURL+"/api/v1/nostr/auth", "application/json", strings.NewReader(string(reqJSON)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("auth failed: %s", string(body))
	}

	return nil
}

func runInteractiveTerminal(relayURL, npub string) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("nostr-ssh> ")
		if !scanner.Scan() {
			break
		}

		command := strings.TrimSpace(scanner.Text())
		parts := strings.Fields(command)

		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "list":
			handleList(relayURL, npub)
		case "add":
			handleAdd(relayURL, npub, scanner)
		case "remove":
			if len(parts) < 2 {
				fmt.Println("Usage: remove <key-name>")
				continue
			}
			handleRemove(relayURL, npub, parts[1])
		case "help":
			handleHelp()
		case "quit", "exit":
			fmt.Println("Goodbye!")
			return
		default:
			fmt.Println("Unknown command. Type 'help' for available commands.")
		}
	}
}

func handleList(relayURL, npub string) {
	fmt.Println("📋 Listing SSH keys...")

	req, err := http.NewRequest("GET", relayURL+"/api/v1/ssh-keys", nil)
	if err != nil {
		fmt.Printf("❌ Error creating request: %v\n", err)
		return
	}

	req.Header.Set("X-Nostr-Pubkey", npub)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ Error making request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("❌ Error: %s\n", string(body))
		return
	}

	var result struct {
		Success bool `json:"success"`
		Keys    []struct {
			Name      string `json:"name"`
			Type      string `json:"type"`
			CreatedAt string `json:"created_at"`
			Comment   string `json:"comment"`
			OwnerNpub string `json:"owner_npub"`
		} `json:"keys"`
		Count int `json:"count"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("❌ Error decoding response: %v\n", err)
		return
	}

	if !result.Success {
		fmt.Println("❌ Failed to list keys")
		return
	}

	if result.Count == 0 {
		fmt.Println("📝 No SSH keys found")
		return
	}

	fmt.Printf("📋 Found %d SSH key(s):\n", result.Count)
	for _, key := range result.Keys {
		fmt.Printf("  🔑 %s (%s) - Created: %s\n", key.Name, key.Type, key.CreatedAt)
		if key.Comment != "" {
			fmt.Printf("      Comment: %s\n", key.Comment)
		}
	}
}

func handleAdd(relayURL, npub string, scanner *bufio.Scanner) {
	fmt.Println("➕ Adding SSH key...")

	fmt.Print("Key name: ")
	scanner.Scan()
	name := strings.TrimSpace(scanner.Text())
	if name == "" {
		fmt.Println("❌ Key name cannot be empty")
		return
	}

	fmt.Print("Private key (PEM format): ")
	scanner.Scan()
	privateKey := strings.TrimSpace(scanner.Text())
	if privateKey == "" {
		fmt.Println("❌ Private key cannot be empty")
		return
	}

	fmt.Print("Public key (optional): ")
	scanner.Scan()
	publicKey := strings.TrimSpace(scanner.Text())

	fmt.Print("Description (optional): ")
	scanner.Scan()
	description := strings.TrimSpace(scanner.Text())

	// Create request
	reqBody := map[string]string{
		"name":        name,
		"private_key": privateKey,
		"public_key":  publicKey,
		"description": description,
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Printf("❌ Error creating request: %v\n", err)
		return
	}

	req, err := http.NewRequest("POST", relayURL+"/api/v1/ssh-keys", strings.NewReader(string(reqJSON)))
	if err != nil {
		fmt.Printf("❌ Error creating request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Nostr-Pubkey", npub)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ Error making request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		fmt.Printf("✅ SSH key '%s' added successfully\n", name)
	} else {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("❌ Error: %s\n", string(body))
	}
}

func handleRemove(relayURL, npub, keyName string) {
	fmt.Printf("🗑️  Removing SSH key '%s'...\n", keyName)

	req, err := http.NewRequest("DELETE", relayURL+"/api/v1/ssh-keys/"+keyName, nil)
	if err != nil {
		fmt.Printf("❌ Error creating request: %v\n", err)
		return
	}

	req.Header.Set("X-Nostr-Pubkey", npub)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ Error making request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("✅ SSH key '%s' removed successfully\n", keyName)
	} else {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("❌ Error: %s\n", string(body))
	}
}

func handleHelp() {
	fmt.Println("📖 Available commands:")
	fmt.Println("  list                    - List your SSH keys")
	fmt.Println("  add                     - Add a new SSH key")
	fmt.Println("  remove <key-name>       - Remove an SSH key")
	fmt.Println("  help                    - Show this help")
	fmt.Println("  quit/exit               - Exit the program")
	fmt.Println()
	fmt.Println("🔐 Authentication:")
	fmt.Println("  Set MERCURY_PRIVATE_KEY environment variable with your Nostr private key")
	fmt.Println("  Example: export MERCURY_PRIVATE_KEY=\"nsec1your-private-key\"")
}

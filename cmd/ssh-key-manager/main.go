package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mercury-relay/internal/config"
	"mercury-relay/internal/transport"
)

func main() {
	fmt.Println("Mercury Relay SSH Key Manager")
	fmt.Println("=============================")

	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create SSH transport for key management
	sshTransport := transport.NewSSHTransport(cfg.SSH)

	// Initialize key manager
	if err := sshTransport.Start(nil); err != nil {
		log.Printf("Warning: SSH transport initialization failed: %v", err)
	}

	// Start interactive terminal
	runInteractiveTerminal(sshTransport)
}

func runInteractiveTerminal(sshTransport *transport.SSHTransport) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("ssh-key-manager> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		command := parts[0]
		args := parts[1:]

		switch command {
		case "help", "h":
			showHelp()
		case "list", "ls":
			handleListKeys(sshTransport)
		case "add", "generate":
			handleAddKey(sshTransport, args)
		case "remove", "rm":
			handleRemoveKey(sshTransport, args)
		case "show":
			handleShowKey(sshTransport, args)
		case "test":
			handleTestConnection(sshTransport)
		case "quit", "exit", "q":
			fmt.Println("Goodbye!")
			return
		default:
			fmt.Printf("Unknown command: %s. Type 'help' for available commands.\n", command)
		}
	}
}

func showHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  help, h           - Show this help message")
	fmt.Println("  list, ls          - List all SSH keys")
	fmt.Println("  add <name>        - Generate a new SSH key pair")
	fmt.Println("  remove <name>     - Remove an SSH key pair")
	fmt.Println("  show <name>       - Show details of a specific key")
	fmt.Println("  test              - Test SSH connection")
	fmt.Println("  quit, exit, q     - Exit the program")
}

func handleListKeys(sshTransport *transport.SSHTransport) {
	// This would need to be implemented in the SSH transport
	// For now, we'll show a placeholder
	fmt.Println("SSH Key listing functionality would be implemented here")
	fmt.Println("This would show all available SSH keys with their details")
}

func handleAddKey(sshTransport *transport.SSHTransport, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: add <key-name>")
		return
	}

	keyName := args[0]

	// Validate key name
	if !isValidKeyName(keyName) {
		fmt.Println("Invalid key name. Use only alphanumeric characters and hyphens.")
		return
	}

	// Check if key already exists
	// This would need to be implemented in the SSH transport
	fmt.Printf("Generating SSH key pair: %s\n", keyName)
	fmt.Println("Key generation functionality would be implemented here")
}

func handleRemoveKey(sshTransport *transport.SSHTransport, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: remove <key-name>")
		return
	}

	keyName := args[0]

	// Confirm deletion
	fmt.Printf("Are you sure you want to delete key '%s'? (y/N): ", keyName)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return
	}

	response := strings.ToLower(strings.TrimSpace(scanner.Text()))
	if response != "y" && response != "yes" {
		fmt.Println("Operation cancelled.")
		return
	}

	fmt.Printf("Removing SSH key: %s\n", keyName)
	fmt.Println("Key removal functionality would be implemented here")
}

func handleShowKey(sshTransport *transport.SSHTransport, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: show <key-name>")
		return
	}

	keyName := args[0]
	fmt.Printf("Showing details for key: %s\n", keyName)
	fmt.Println("Key details functionality would be implemented here")
}

func handleTestConnection(sshTransport *transport.SSHTransport) {
	fmt.Println("Testing SSH connection...")

	// This would test the SSH connection
	fmt.Println("SSH connection test functionality would be implemented here")

	if sshTransport.IsHealthy() {
		fmt.Println("✓ SSH transport is healthy")
	} else {
		fmt.Println("✗ SSH transport is not healthy")
	}
}

func isValidKeyName(name string) bool {
	if len(name) == 0 || len(name) > 50 {
		return false
	}

	for _, char := range name {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_') {
			return false
		}
	}

	return true
}

// Additional utility functions for key management

func createKeyDirectory(keyDir string) error {
	return os.MkdirAll(keyDir, 0700)
}

func keyExists(keyName, keyDir, keyExt string) bool {
	keyPath := filepath.Join(keyDir, keyName+keyExt)
	_, err := os.Stat(keyPath)
	return !os.IsNotExist(err)
}

func getKeyInfo(keyName, keyDir string) (os.FileInfo, error) {
	keyPath := filepath.Join(keyDir, keyName+".pem")
	return os.Stat(keyPath)
}

func formatKeySize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

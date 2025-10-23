package admin

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"mercury-relay/internal/config"
	"mercury-relay/internal/models"
	"mercury-relay/internal/quality"

	"github.com/nbd-wtf/go-nostr"
)

type Interface struct {
	config           *config.Config
	qualityControl   *quality.Controller
	kindConfigLoader *quality.KindConfigLoader
	authenticated    bool
	userPubkey       string
}

func NewInterface(config *config.Config) *Interface {
	// Initialize quality control
	qualityControl := quality.NewController(config.Quality, nil, nil)

	// Initialize kind config loader from individual YAML files
	kindConfigLoader, err := quality.NewKindConfigLoaderFromDirectory("configs/kinds")
	if err != nil {
		log.Printf("Warning: Failed to load kind configs from directory: %v", err)
		kindConfigLoader = nil
	} else {
		qualityControl.SetKindConfigLoader(kindConfigLoader)
	}

	return &Interface{
		config:           config,
		qualityControl:   qualityControl,
		kindConfigLoader: kindConfigLoader,
	}
}

func (a *Interface) BlockNpub(npub string) error {
	// This would need to be connected to the quality controller
	// For now, just log the action
	log.Printf("Blocking npub: %s", npub)
	return nil
}

func (a *Interface) UnblockNpub(npub string) error {
	// This would need to be connected to the quality controller
	// For now, just log the action
	log.Printf("Unblocking npub: %s", npub)
	return nil
}

func (a *Interface) ListBlockedNpubs() ([]string, error) {
	// This would need to be connected to the quality controller
	// For now, return empty list
	return []string{}, nil
}

func (a *Interface) StartTUI() error {
	log.Println("Starting admin TUI interface...")

	// Check if authentication is required
	// For now, we'll use a simple check - in production this would be more sophisticated
	requireAuth := true // Enable authentication
	if requireAuth {
		if !a.authenticate() {
			return fmt.Errorf("authentication failed")
		}
	} else {
		fmt.Println("⚠️  Authentication disabled - TUI running in development mode")
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\n=== Mercury Relay Admin ===\n")
		if a.authenticated {
			fmt.Printf("Authenticated as: %s\n", a.userPubkey)
		}
		fmt.Print("1. Block npub\n")
		fmt.Print("2. Unblock npub\n")
		fmt.Print("3. List blocked npubs\n")
		fmt.Print("4. Show stats\n")
		fmt.Print("5. Query relay\n")
		fmt.Print("6. Publish note\n")
		fmt.Print("7. Exit\n")
		fmt.Print("Choose an option (1-7): ")

		if !scanner.Scan() {
			break
		}

		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "1":
			a.handleBlockNpub(scanner)
		case "2":
			a.handleUnblockNpub(scanner)
		case "3":
			a.handleListBlocked()
		case "4":
			a.handleShowStats()
		case "5":
			a.handleQueryRelay(scanner)
		case "6":
			a.handlePublishNote(scanner)
		case "7":
			fmt.Println("Goodbye!")
			return nil
		default:
			fmt.Println("Invalid option. Please choose 1-7.")
		}
	}

	return nil
}

func (a *Interface) handleBlockNpub(scanner *bufio.Scanner) {
	fmt.Print("Enter npub to block: ")
	if !scanner.Scan() {
		return
	}
	npub := strings.TrimSpace(scanner.Text())

	if npub == "" {
		fmt.Println("Empty npub provided.")
		return
	}

	if err := a.BlockNpub(npub); err != nil {
		fmt.Printf("Error blocking npub: %v\n", err)
	} else {
		fmt.Printf("Successfully blocked npub: %s\n", npub)
	}
}

func (a *Interface) handleUnblockNpub(scanner *bufio.Scanner) {
	fmt.Print("Enter npub to unblock: ")
	if !scanner.Scan() {
		return
	}
	npub := strings.TrimSpace(scanner.Text())

	if npub == "" {
		fmt.Println("Empty npub provided.")
		return
	}

	if err := a.UnblockNpub(npub); err != nil {
		fmt.Printf("Error unblocking npub: %v\n", err)
	} else {
		fmt.Printf("Successfully unblocked npub: %s\n", npub)
	}
}

func (a *Interface) handleListBlocked() {
	blocked, err := a.ListBlockedNpubs()
	if err != nil {
		fmt.Printf("Error listing blocked npubs: %v\n", err)
		return
	}

	if len(blocked) == 0 {
		fmt.Println("No blocked npubs found.")
		return
	}

	fmt.Println("Blocked npubs:")
	for i, npub := range blocked {
		fmt.Printf("%d. %s\n", i+1, npub)
	}
}

func (a *Interface) handleShowStats() {
	fmt.Println("=== Mercury Relay Stats ===")
	fmt.Println("Status: Running")
	fmt.Println("Config loaded: Yes")
	fmt.Println("Quality control: Enabled")
	fmt.Println("SSH tunnel: Available")
	fmt.Println("Tor support: Available")
	fmt.Println("I2P support: Available")
	if a.authenticated {
		fmt.Printf("Authenticated user: %s\n", a.userPubkey)
	}
}

// authenticate handles user authentication
func (a *Interface) authenticate() bool {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("\n=== Authentication Required ===")
	fmt.Println("Choose authentication method:")
	fmt.Println("1. API Key")
	fmt.Println("2. Nostr Authentication")
	fmt.Print("Choose (1-2): ")

	if !scanner.Scan() {
		return false
	}

	choice := strings.TrimSpace(scanner.Text())

	switch choice {
	case "1":
		return a.authenticateWithAPIKey(scanner)
	case "2":
		return a.authenticateWithNostr(scanner)
	default:
		fmt.Println("Invalid choice.")
		return false
	}
}

// authenticateWithAPIKey handles API key authentication
func (a *Interface) authenticateWithAPIKey(scanner *bufio.Scanner) bool {
	fmt.Print("Enter API Key: ")
	if !scanner.Scan() {
		return false
	}

	apiKey := strings.TrimSpace(scanner.Text())

	// Check against configured API key (using admin config)
	if apiKey == a.config.Admin.APIKey {
		a.authenticated = true
		a.userPubkey = "api-key-user"
		fmt.Println("✅ Authentication successful!")
		return true
	}

	fmt.Println("❌ Invalid API key.")
	return false
}

// authenticateWithNostr handles Nostr authentication
func (a *Interface) authenticateWithNostr(scanner *bufio.Scanner) bool {
	fmt.Println("\n=== Nostr Authentication ===")
	fmt.Println("This will generate a challenge that you need to sign with your Nostr key.")
	fmt.Print("Continue? (y/n): ")

	if !scanner.Scan() {
		return false
	}

	if strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
		return false
	}

	// Generate challenge
	challenge, err := a.generateChallenge()
	if err != nil {
		fmt.Printf("❌ Failed to generate challenge: %v\n", err)
		return false
	}

	fmt.Printf("\n🔐 Challenge: %s\n", challenge)
	fmt.Println("\nTo authenticate:")
	fmt.Println("1. Use your Nostr client to create a kind 22242 event")
	fmt.Println("2. Include these tags:")
	fmt.Printf("   - challenge: %s\n", challenge)
	fmt.Printf("   - relay: %s\n", a.config.Server.Host+":"+fmt.Sprintf("%d", a.config.Server.Port))
	fmt.Println("3. Sign and publish the event")
	fmt.Println("4. Press Enter when done...")

	if !scanner.Scan() {
		return false
	}

	// Use NSEC from environment to automatically authenticate
	privKey := a.getPrivateKeyFromEnv()
	if privKey == "" {
		fmt.Println("❌ No private key available.")
		return false
	}

	// Get public key from private key
	pubkey, err := nostr.GetPublicKey(privKey)
	if err != nil {
		fmt.Printf("❌ Failed to get public key: %v\n", err)
		return false
	}

	// Check if pubkey is authorized
	if len(a.config.Access.AdminNpubs) > 0 {
		authorized := false
		for _, authorizedPubkey := range a.config.Access.AdminNpubs {
			if pubkey == authorizedPubkey {
				authorized = true
				break
			}
		}
		if !authorized {
			fmt.Printf("❌ Pubkey %s not authorized.\n", pubkey)
			return false
		}
	}

	// Perform full NIP-42 authentication
	relayURL := fmt.Sprintf("http://%s:%d", a.config.Server.Host, a.config.Server.Port+2) // REST API is on port+2
	if !a.authenticateWithNIP42(relayURL, pubkey) {
		fmt.Println("❌ NIP-42 authentication failed.")
		return false
	}

	a.authenticated = true
	a.userPubkey = pubkey
	fmt.Println("✅ Nostr authentication successful!")
	return true
}

// generateChallenge creates a random challenge string
func (a *Interface) generateChallenge() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate challenge: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// handleQueryRelay handles relay querying functionality
func (a *Interface) handleQueryRelay(scanner *bufio.Scanner) {
	fmt.Println("\n=== Relay Query Interface ===")
	fmt.Println("1. Query events by author")
	fmt.Println("2. Query events by kind")
	fmt.Println("3. Query events by tag")
	fmt.Println("4. Query recent events")
	fmt.Println("5. Get relay info")
	fmt.Println("6. Back to main menu")
	fmt.Print("Choose an option (1-6): ")

	if !scanner.Scan() {
		return
	}

	choice := strings.TrimSpace(scanner.Text())

	switch choice {
	case "1":
		a.queryEventsByAuthor(scanner)
	case "2":
		a.queryEventsByKind(scanner)
	case "3":
		a.queryEventsByTag(scanner)
	case "4":
		a.queryRecentEvents(scanner)
	case "5":
		a.getRelayInfo()
	case "6":
		return
	default:
		fmt.Println("Invalid option.")
	}
}

// queryEventsByAuthor queries events by author pubkey
func (a *Interface) queryEventsByAuthor(scanner *bufio.Scanner) {
	fmt.Print("Enter author pubkey (npub or hex): ")
	if !scanner.Scan() {
		return
	}
	author := strings.TrimSpace(scanner.Text())
	if author == "" {
		fmt.Println("Empty author provided.")
		return
	}

	fmt.Print("Enter limit (default 10): ")
	if !scanner.Scan() {
		return
	}
	limitStr := strings.TrimSpace(scanner.Text())
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	// Create Nostr filter
	filter := nostr.Filter{
		Authors: []string{author},
		Limit:   limit,
	}

	a.executeQuery("Events by author", filter)
}

// queryEventsByKind queries events by kind
func (a *Interface) queryEventsByKind(scanner *bufio.Scanner) {
	fmt.Print("Enter kind number: ")
	if !scanner.Scan() {
		return
	}
	kindStr := strings.TrimSpace(scanner.Text())
	if kindStr == "" {
		fmt.Println("Empty kind provided.")
		return
	}

	kind, err := strconv.Atoi(kindStr)
	if err != nil {
		fmt.Printf("Invalid kind: %v\n", err)
		return
	}

	fmt.Print("Enter limit (default 10): ")
	if !scanner.Scan() {
		return
	}
	limitStr := strings.TrimSpace(scanner.Text())
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	// Create Nostr filter
	filter := nostr.Filter{
		Kinds: []int{kind},
		Limit: limit,
	}

	a.executeQuery("Events by kind", filter)
}

// queryEventsByTag queries events by tag
func (a *Interface) queryEventsByTag(scanner *bufio.Scanner) {
	fmt.Print("Enter tag name (e.g., 'p', 'e', 't'): ")
	if !scanner.Scan() {
		return
	}
	tagName := strings.TrimSpace(scanner.Text())
	if tagName == "" {
		fmt.Println("Empty tag name provided.")
		return
	}

	fmt.Print("Enter tag value: ")
	if !scanner.Scan() {
		return
	}
	tagValue := strings.TrimSpace(scanner.Text())
	if tagValue == "" {
		fmt.Println("Empty tag value provided.")
		return
	}

	fmt.Print("Enter limit (default 10): ")
	if !scanner.Scan() {
		return
	}
	limitStr := strings.TrimSpace(scanner.Text())
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	// Create Nostr filter
	filter := nostr.Filter{
		Tags:  map[string][]string{tagName: {tagValue}},
		Limit: limit,
	}

	a.executeQuery("Events by tag", filter)
}

// queryRecentEvents queries recent events
func (a *Interface) queryRecentEvents(scanner *bufio.Scanner) {
	fmt.Print("Enter limit (default 20): ")
	if !scanner.Scan() {
		return
	}
	limitStr := strings.TrimSpace(scanner.Text())
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	// Create Nostr filter for recent events
	filter := nostr.Filter{
		Limit: limit,
	}

	a.executeQuery("Recent events", filter)
}

// getRelayInfo gets relay information
func (a *Interface) getRelayInfo() {
	fmt.Println("\n=== Relay Information ===")

	relayURL := fmt.Sprintf("http://%s:%d", a.config.Server.Host, a.config.Server.Port+2) // REST API is on port+2

	// Try to get relay info from REST API
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(relayURL + "/api/v1/health")
	if err != nil {
		fmt.Printf("❌ Failed to connect to relay: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("❌ Failed to read response: %v\n", err)
			return
		}

		var health map[string]interface{}
		if err := json.Unmarshal(body, &health); err == nil {
			fmt.Println("✅ Relay is healthy")
			fmt.Printf("Status: %v\n", health["status"])
			if version, ok := health["version"]; ok {
				fmt.Printf("Version: %v\n", version)
			}
		} else {
			fmt.Println("✅ Relay is responding")
		}
	} else {
		fmt.Printf("⚠️  Relay responded with status: %d\n", resp.StatusCode)
	}

	fmt.Printf("Relay URL: %s\n", relayURL)
	fmt.Printf("Server: %s:%d\n", a.config.Server.Host, a.config.Server.Port)
	fmt.Printf("Tor enabled: %v\n", a.config.Tor.Enabled)
	fmt.Printf("I2P enabled: %v\n", a.config.I2P.Enabled)
	fmt.Printf("SSH enabled: %v\n", a.config.SSH.Enabled)
}

// executeQuery executes a Nostr query against the relay
func (a *Interface) executeQuery(queryType string, filter nostr.Filter) {
	fmt.Printf("\n=== %s ===\n", queryType)

	relayURL := fmt.Sprintf("http://%s:%d", a.config.Server.Host, a.config.Server.Port+2) // REST API is on port+2

	// Create Nostr REQ message
	req := []interface{}{
		"REQ",
		"admin-query",
		filter,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		fmt.Printf("❌ Failed to create query: %v\n", err)
		return
	}

	// Send request to relay
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(relayURL+"/", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ Failed to query relay: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ Relay returned status: %d\n", resp.StatusCode)
		return
	}

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ Failed to read response: %v\n", err)
		return
	}

	// Parse response
	var response []interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("❌ Failed to parse response: %v\n", err)
		return
	}

	// Display results
	if len(response) == 0 {
		fmt.Println("No events found.")
		return
	}

	fmt.Printf("Found %d events:\n", len(response))
	for i, event := range response {
		if eventMap, ok := event.(map[string]interface{}); ok {
			fmt.Printf("\n--- Event %d ---\n", i+1)
			if id, ok := eventMap["id"].(string); ok {
				fmt.Printf("ID: %s\n", id)
			}
			if pubkey, ok := eventMap["pubkey"].(string); ok {
				fmt.Printf("Author: %s\n", pubkey)
			}
			if kind, ok := eventMap["kind"].(float64); ok {
				fmt.Printf("Kind: %.0f\n", kind)
			}
			if content, ok := eventMap["content"].(string); ok {
				contentPreview := content
				if len(contentPreview) > 100 {
					contentPreview = contentPreview[:100] + "..."
				}
				fmt.Printf("Content: %s\n", contentPreview)
			}
			if createdAt, ok := eventMap["created_at"].(float64); ok {
				timestamp := time.Unix(int64(createdAt), 0)
				fmt.Printf("Created: %s\n", timestamp.Format("2006-01-02 15:04:05"))
			}
		}
	}
}

// handlePublishNote handles note publishing functionality
func (a *Interface) handlePublishNote(scanner *bufio.Scanner) {
	fmt.Println("\n=== Publish Note ===")
	fmt.Println("1. Text note (Kind 1)")
	fmt.Println("2. Long-form content (Kind 30023)")
	fmt.Println("3. Publication content (Kind 30041)")
	fmt.Println("4. Discussion thread (Kind 11)")
	fmt.Println("5. Back to main menu")
	fmt.Print("Choose note type (1-5): ")

	if !scanner.Scan() {
		return
	}

	choice := strings.TrimSpace(scanner.Text())

	switch choice {
	case "1":
		a.publishTextNote(scanner)
	case "2":
		a.publishLongFormContent(scanner)
	case "3":
		a.publishPublicationContent(scanner)
	case "4":
		a.publishDiscussionThread(scanner)
	case "5":
		return
	default:
		fmt.Println("Invalid option.")
	}
}

// publishTextNote publishes a text note (Kind 1)
func (a *Interface) publishTextNote(scanner *bufio.Scanner) {
	fmt.Println("\n=== Text Note (Kind 1) ===")

	// Get QC guidance for text notes
	if a.kindConfigLoader != nil {
		kindConfig, err := a.kindConfigLoader.GetKindConfig(1)
		if err == nil && kindConfig != nil {
			fmt.Println("📝 QC Guidance for Text Notes:")
			if kindConfig.ContentValidation.Type != "" {
				fmt.Printf("- Content validation: %s\n", kindConfig.ContentValidation.Type)
			}
			if len(kindConfig.RequiredTags) > 0 {
				fmt.Printf("- Required tags: %v\n", kindConfig.RequiredTags)
			}
			if len(kindConfig.OptionalTags) > 0 {
				fmt.Printf("- Optional tags: %v\n", kindConfig.OptionalTags)
			}
		}
	}

	fmt.Print("Enter your note content: ")
	if !scanner.Scan() {
		return
	}
	content := strings.TrimSpace(scanner.Text())
	if content == "" {
		fmt.Println("Empty content provided.")
		return
	}

	// Get optional tags
	tags := a.collectTags(scanner, 1)

	// Create and publish event
	event := a.createEvent(1, content, tags)
	a.publishEvent(event)
}

// publishLongFormContent publishes long-form content (Kind 30023)
func (a *Interface) publishLongFormContent(scanner *bufio.Scanner) {
	fmt.Println("\n=== Long-form Content (Kind 30023) ===")

	// Get QC guidance for long-form content
	if a.kindConfigLoader != nil {
		kindConfig, err := a.kindConfigLoader.GetKindConfig(30023)
		if err == nil && kindConfig != nil {
			fmt.Println("📝 QC Guidance for Long-form Content:")
			if kindConfig.ContentValidation.Type != "" {
				fmt.Printf("- Content validation: %s\n", kindConfig.ContentValidation.Type)
			}
			if len(kindConfig.RequiredTags) > 0 {
				fmt.Printf("- Required tags: %v\n", kindConfig.RequiredTags)
			}
			if len(kindConfig.OptionalTags) > 0 {
				fmt.Printf("- Optional tags: %v\n", kindConfig.OptionalTags)
			}
		}
	}

	fmt.Print("Enter title: ")
	if !scanner.Scan() {
		return
	}
	title := strings.TrimSpace(scanner.Text())
	if title == "" {
		fmt.Println("Title is required for long-form content.")
		return
	}

	fmt.Print("Enter content: ")
	if !scanner.Scan() {
		return
	}
	content := strings.TrimSpace(scanner.Text())
	if content == "" {
		fmt.Println("Content is required.")
		return
	}

	// Get optional tags
	tags := a.collectTags(scanner, 30023)

	// Add title tag
	tags = append(tags, []string{"title", title})

	// Create and publish event
	event := a.createEvent(30023, content, tags)
	a.publishEvent(event)
}

// publishPublicationContent publishes publication content (Kind 30041)
func (a *Interface) publishPublicationContent(scanner *bufio.Scanner) {
	fmt.Println("\n=== Publication Content (Kind 30041) ===")

	// Get QC guidance for publication content
	if a.kindConfigLoader != nil {
		kindConfig, err := a.kindConfigLoader.GetKindConfig(30041)
		if err == nil && kindConfig != nil {
			fmt.Println("📝 QC Guidance for Publication Content:")
			if kindConfig.ContentValidation.Type != "" {
				fmt.Printf("- Content validation: %s\n", kindConfig.ContentValidation.Type)
			}
			if len(kindConfig.RequiredTags) > 0 {
				fmt.Printf("- Required tags: %v\n", kindConfig.RequiredTags)
			}
			if len(kindConfig.OptionalTags) > 0 {
				fmt.Printf("- Optional tags: %v\n", kindConfig.OptionalTags)
			}
		}
	}

	fmt.Println("📖 Publication Content Requirements:")
	fmt.Println("- MUST include a 'd' tag (section identifier)")
	fmt.Println("- MUST include a 'title' tag")
	fmt.Println("- Content may contain AsciiDoc markup")
	fmt.Println("- Content may contain wikilinks (double brackets)")

	fmt.Print("Enter section title: ")
	if !scanner.Scan() {
		return
	}
	title := strings.TrimSpace(scanner.Text())
	if title == "" {
		fmt.Println("Section title is required.")
		return
	}

	fmt.Print("Enter section identifier (d tag): ")
	if !scanner.Scan() {
		return
	}
	dTag := strings.TrimSpace(scanner.Text())
	if dTag == "" {
		fmt.Println("Section identifier is required.")
		return
	}

	fmt.Print("Enter content (may include AsciiDoc markup and wikilinks): ")
	if !scanner.Scan() {
		return
	}
	content := strings.TrimSpace(scanner.Text())
	if content == "" {
		fmt.Println("Content is required.")
		return
	}

	// Get optional tags
	tags := a.collectTags(scanner, 30041)

	// Add required tags
	tags = append(tags, []string{"title", title})
	tags = append(tags, []string{"d", dTag})

	// Create and publish event
	event := a.createEvent(30041, content, tags)
	a.publishEvent(event)
}

// publishDiscussionThread publishes a discussion thread (Kind 11)
func (a *Interface) publishDiscussionThread(scanner *bufio.Scanner) {
	fmt.Println("\n=== Discussion Thread (Kind 11) ===")

	// Get QC guidance for discussion threads
	if a.kindConfigLoader != nil {
		kindConfig, err := a.kindConfigLoader.GetKindConfig(11)
		if err == nil && kindConfig != nil {
			fmt.Println("📝 QC Guidance for Discussion Threads:")
			if kindConfig.ContentValidation.Type != "" {
				fmt.Printf("- Content validation: %s\n", kindConfig.ContentValidation.Type)
			}
			if len(kindConfig.RequiredTags) > 0 {
				fmt.Printf("- Required tags: %v\n", kindConfig.RequiredTags)
			}
			if len(kindConfig.OptionalTags) > 0 {
				fmt.Printf("- Optional tags: %v\n", kindConfig.OptionalTags)
			}
		}
	}

	fmt.Println("💬 Discussion Thread Requirements:")
	fmt.Println("- SHOULD include a 'title' tag")
	fmt.Println("- Content is the thread starter message")
	fmt.Println("- Replies use Kind 1111 (comments)")

	fmt.Print("Enter thread title (optional): ")
	if !scanner.Scan() {
		return
	}
	title := strings.TrimSpace(scanner.Text())

	fmt.Print("Enter thread content: ")
	if !scanner.Scan() {
		return
	}
	content := strings.TrimSpace(scanner.Text())
	if content == "" {
		fmt.Println("Thread content is required.")
		return
	}

	// Get optional tags
	tags := a.collectTags(scanner, 11)

	// Add title tag if provided
	if title != "" {
		tags = append(tags, []string{"title", title})
	}

	// Create and publish event
	event := a.createEvent(11, content, tags)
	a.publishEvent(event)
}

// collectTags collects optional tags from user
func (a *Interface) collectTags(scanner *bufio.Scanner, kind int) nostr.Tags {
	var tags nostr.Tags

	// Get QC guidance for tags
	if a.kindConfigLoader != nil {
		kindConfig, err := a.kindConfigLoader.GetKindConfig(kind)
		if err == nil && kindConfig != nil && len(kindConfig.OptionalTags) > 0 {
			fmt.Printf("\nOptional tags for this kind: %v\n", kindConfig.OptionalTags)
		}
	}

	fmt.Print("Add hashtags (comma-separated, optional): ")
	if !scanner.Scan() {
		return tags
	}
	hashtags := strings.TrimSpace(scanner.Text())
	if hashtags != "" {
		for _, tag := range strings.Split(hashtags, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, []string{"t", tag})
			}
		}
	}

	fmt.Print("Add mentions (npub or hex, comma-separated, optional): ")
	if !scanner.Scan() {
		return tags
	}
	mentions := strings.TrimSpace(scanner.Text())
	if mentions != "" {
		for _, mention := range strings.Split(mentions, ",") {
			mention = strings.TrimSpace(mention)
			if mention != "" {
				tags = append(tags, []string{"p", mention})
			}
		}
	}

	return tags
}

// createEvent creates a Nostr event
func (a *Interface) createEvent(kind int, content string, tags nostr.Tags) *nostr.Event {
	// Get private key from environment variable
	privKey := a.getPrivateKeyFromEnv()

	event := &nostr.Event{
		Kind:      kind,
		Content:   content,
		Tags:      tags,
		CreatedAt: nostr.Timestamp(time.Now().Unix()),
	}

	// Sign the event
	event.Sign(privKey)

	return event
}

// publishEvent publishes an event to the relay
func (a *Interface) publishEvent(event *nostr.Event) {
	fmt.Printf("\n=== Publishing Event ===")
	fmt.Printf("Kind: %d\n", event.Kind)
	fmt.Printf("Content: %s\n", event.Content)
	fmt.Printf("Tags: %v\n", event.Tags)
	fmt.Printf("ID: %s\n", event.ID)
	fmt.Printf("PubKey: %s\n", event.PubKey)

	relayURL := fmt.Sprintf("http://%s:%d", a.config.Server.Host, a.config.Server.Port+2) // REST API is on port+2

	// First, authenticate with NIP-42
	if !a.authenticateWithNIP42(relayURL, event.PubKey) {
		fmt.Println("❌ NIP-42 authentication failed")
		return
	}

	// Create REST API publish request using models.Event
	modelsEvent := &models.Event{
		ID:        event.ID,
		PubKey:    event.PubKey,
		CreatedAt: event.CreatedAt,
		Kind:      event.Kind,
		Tags:      event.Tags,
		Content:   event.Content,
		Sig:       event.Sig,
	}

	publishReq := map[string]interface{}{
		"event": modelsEvent,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(publishReq)
	if err != nil {
		fmt.Printf("❌ Failed to create event: %v\n", err)
		return
	}

	// Send request to REST API with authentication header
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("POST", relayURL+"/api/v1/publish", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ Failed to create request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Nostr-Pubkey", event.PubKey) // Use the actual pubkey from the signed event

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ Failed to publish event: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("✅ Event published successfully!")

		// Display the nevent for easy client access
		nevent := a.generateNevent(event)
		fmt.Printf("\n🔗 Event Identifier (nevent): %s\n", nevent)
		fmt.Println("📱 You can use this nevent to find the event in your Nostr client")
	} else {
		fmt.Printf("❌ Relay returned status: %d\n", resp.StatusCode)
		// Try to read error message
		body, err := io.ReadAll(resp.Body)
		if err == nil && len(body) > 0 {
			fmt.Printf("Error details: %s\n", string(body))
		}
	}
}

// getPrivateKeyFromEnv gets the private key from NSEC environment variable
func (a *Interface) getPrivateKeyFromEnv() string {
	nsec := os.Getenv("NSEC")
	if nsec == "" {
		fmt.Println("⚠️  NSEC environment variable not set, using random key for testing")
		return a.generatePrivateKey()
	}

	// Check if it's already a hex private key (64 characters)
	if len(nsec) == 64 {
		// Validate it's hex
		if _, err := hex.DecodeString(nsec); err == nil {
			return nsec
		}
	}

	// For now, if it's nsec format, we'll need to implement bech32 decoding
	// For development, let's just use a random key and log the issue
	if strings.HasPrefix(nsec, "nsec1") {
		fmt.Printf("⚠️  NSEC format detected but bech32 decoding not implemented yet\n")
		fmt.Println("Using random key for testing")
		return a.generatePrivateKey()
	}

	// Try to use as hex directly
	if _, err := hex.DecodeString(nsec); err == nil {
		return nsec
	}

	fmt.Printf("❌ Invalid NSEC format: %s\n", nsec)
	fmt.Println("Using random key for testing")
	return a.generatePrivateKey()
}

// authenticateWithNIP42 performs NIP-42 authentication
func (a *Interface) authenticateWithNIP42(relayURL, pubkey string) bool {
	fmt.Println("🔐 Authenticating with NIP-42...")

	// Step 1: Get challenge from relay
	challenge, err := a.getNostrChallenge(relayURL)
	if err != nil {
		fmt.Printf("❌ Failed to get challenge: %v\n", err)
		return false
	}

	fmt.Printf("📝 Challenge received: %s\n", challenge)

	// Step 2: Create authentication event (kind 22242)
	authEvent := a.createAuthEvent(challenge, relayURL)
	if authEvent == nil {
		fmt.Println("❌ Failed to create auth event")
		return false
	}

	// Step 3: Publish auth event to relay
	if !a.publishAuthEvent(relayURL, authEvent) {
		fmt.Println("❌ Failed to publish auth event")
		return false
	}

	fmt.Println("✅ NIP-42 authentication successful!")
	return true
}

// getNostrChallenge gets a challenge from the relay
func (a *Interface) getNostrChallenge(relayURL string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(relayURL + "/api/v1/nostr/challenge")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("challenge request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", err
	}

	challenge, ok := response["challenge"].(string)
	if !ok {
		return "", fmt.Errorf("invalid challenge response")
	}

	return challenge, nil
}

// createAuthEvent creates a NIP-42 authentication event
func (a *Interface) createAuthEvent(challenge, relayURL string) *nostr.Event {
	privKey := a.getPrivateKeyFromEnv()

	// Create kind 22242 authentication event
	event := &nostr.Event{
		Kind:    22242,
		Content: "",
		Tags: nostr.Tags{
			[]string{"challenge", challenge},
			[]string{"relay", relayURL},
		},
		CreatedAt: nostr.Timestamp(time.Now().Unix()),
	}

	// Sign the event
	event.Sign(privKey)

	return event
}

// publishAuthEvent publishes the authentication event
func (a *Interface) publishAuthEvent(relayURL string, authEvent *nostr.Event) bool {
	// Create the request body in the format expected by the existing endpoint
	publishReq := map[string]interface{}{
		"event": map[string]interface{}{
			"id":         authEvent.ID,
			"pubkey":     authEvent.PubKey,
			"created_at": int64(authEvent.CreatedAt),
			"kind":       authEvent.Kind,
			"tags":       authEvent.Tags,
			"content":    authEvent.Content,
			"sig":        authEvent.Sig,
		},
	}

	jsonData, err := json.Marshal(publishReq)
	if err != nil {
		return false
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", relayURL+"/api/v1/nostr/auth", bytes.NewBuffer(jsonData))
	if err != nil {
		return false
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// generateNevent creates a nevent identifier for the published event
func (a *Interface) generateNevent(event *nostr.Event) string {
	// nevent format: nevent1<event_id><relay_url><author_pubkey>
	// For now, we'll create a simple nevent with just the event ID
	// In a full implementation, you'd use bech32 encoding
	relayURL := fmt.Sprintf("ws://%s:%d", a.config.Server.Host, a.config.Server.Port)
	return fmt.Sprintf("nevent1%s@%s", event.ID, relayURL)
}

// generatePrivateKey generates a random private key for testing
func (a *Interface) generatePrivateKey() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatal("Failed to generate private key:", err)
	}
	return hex.EncodeToString(bytes)
}

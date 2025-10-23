package testgen

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mercury-relay/internal/config"
	"mercury-relay/internal/models"
	"mercury-relay/internal/quality"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/nbd-wtf/go-nostr"
)

type Generator struct {
	config           *config.Config
	rand             *rand.Rand
	kindConfigLoader *quality.KindConfigLoader
	availableKinds   []int
}

type Persona struct {
	Name        string
	PubKey      string
	PrivateKey  string
	Behavior    Behavior
	Topics      []string
	PostingRate float64 // posts per hour
}

type Behavior struct {
	ContentLength [2]int  // min, max
	TagCount      [2]int  // min, max
	ReplyRate     float64 // probability of replying
	RepostRate    float64 // probability of reposting
	ReactionRate  float64 // probability of reacting
}

func NewGenerator(config *config.Config) *Generator {
	generator := &Generator{
		config: config,
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// Load kind configurations from QC system
	generator.loadKindConfigs()

	return generator
}

// loadKindConfigs loads kind configurations from the QC system
func (g *Generator) loadKindConfigs() error {
	// Load kind config loader from the configs directory
	loader, err := quality.NewKindConfigLoader("configs/nostr-event-kinds.yaml")
	if err != nil {
		// If the main config doesn't exist, try to load from individual files
		return g.loadKindConfigsFromFiles()
	}

	g.kindConfigLoader = loader

	// Build list of available kinds from the loader
	// We'll need to implement a method to get available kinds
	g.availableKinds = []int{0, 1, 3, 6, 7, 30023} // Default kinds

	return nil
}

// loadKindConfigsFromFiles loads kind configs from individual YAML files
func (g *Generator) loadKindConfigsFromFiles() error {
	// Load from individual kind files in configs/kinds/
	kindsDir := "configs/kinds"
	files, err := filepath.Glob(filepath.Join(kindsDir, "*.yml"))
	if err != nil {
		return fmt.Errorf("failed to read kinds directory: %w", err)
	}

	var availableKinds []int
	for _, file := range files {
		// Extract kind number from filename (e.g., "0.yml" -> 0)
		filename := filepath.Base(file)
		kindStr := strings.TrimSuffix(filename, ".yml")
		if kind, err := strconv.Atoi(kindStr); err == nil {
			availableKinds = append(availableKinds, kind)
		}
	}

	g.availableKinds = availableKinds
	return nil
}

func (g *Generator) GenerateEvents(count int, personaType string) ([]*models.Event, error) {
	var events []*models.Event

	// Create personas
	personas := g.createPersonas()

	// Select persona based on type
	var selectedPersonas []Persona
	switch personaType {
	case "spammer":
		selectedPersonas = []Persona{personas["spammer"]}
	case "influencer":
		selectedPersonas = []Persona{personas["influencer"]}
	case "casual":
		selectedPersonas = []Persona{personas["casual"]}
	default:
		selectedPersonas = []Persona{personas["casual"], personas["influencer"]}
	}

	// Generate events
	for i := 0; i < count; i++ {
		persona := selectedPersonas[g.rand.Intn(len(selectedPersonas))]
		event := g.generateEvent(persona)
		events = append(events, event)
	}

	return events, nil
}

func (g *Generator) createPersonas() map[string]Persona {
	personas := make(map[string]Persona)

	// Spammer persona
	spammerKey, _ := btcec.NewPrivateKey()
	personas["spammer"] = Persona{
		Name:   "Spammer",
		PubKey: fmt.Sprintf("npub1%x", spammerKey.PubKey().SerializeCompressed()),
		Behavior: Behavior{
			ContentLength: [2]int{10, 50},
			TagCount:      [2]int{5, 20},
			ReplyRate:     0.1,
			RepostRate:    0.05,
			ReactionRate:  0.02,
		},
		Topics:      []string{"crypto", "bitcoin", "nostr", "spam"},
		PostingRate: 10.0, // 10 posts per hour
	}

	// Influencer persona
	influencerKey, _ := btcec.NewPrivateKey()
	personas["influencer"] = Persona{
		Name:   "Influencer",
		PubKey: fmt.Sprintf("npub1%x", influencerKey.PubKey().SerializeCompressed()),
		Behavior: Behavior{
			ContentLength: [2]int{100, 500},
			TagCount:      [2]int{1, 5},
			ReplyRate:     0.3,
			RepostRate:    0.2,
			ReactionRate:  0.4,
		},
		Topics:      []string{"technology", "innovation", "future", "ai"},
		PostingRate: 2.0, // 2 posts per hour
	}

	// Casual user persona
	casualKey, _ := btcec.NewPrivateKey()
	personas["casual"] = Persona{
		Name:   "Casual User",
		PubKey: fmt.Sprintf("npub1%x", casualKey.PubKey().SerializeCompressed()),
		Behavior: Behavior{
			ContentLength: [2]int{20, 200},
			TagCount:      [2]int{0, 3},
			ReplyRate:     0.1,
			RepostRate:    0.05,
			ReactionRate:  0.1,
		},
		Topics:      []string{"life", "thoughts", "random", "daily"},
		PostingRate: 0.5, // 0.5 posts per hour
	}

	return personas
}

func (g *Generator) generateEvent(persona Persona) *models.Event {
	// Select kind first
	kind := g.selectKind(persona)

	// Generate content based on persona behavior and kind
	content := g.generateContentForKind(persona, kind)

	// Generate tags based on kind configuration
	tags := g.generateTags(persona, kind)

	// Create event
	event := &models.Event{
		PubKey:      persona.PubKey,
		CreatedAt:   nostr.Timestamp(time.Now().Add(-time.Duration(g.rand.Intn(3600)) * time.Second).Unix()), // Random time in last hour
		Kind:        kind,
		Tags:        tags,
		Content:     content,
		CreatedAtDB: time.Now(),
	}

	// Generate ID and signature (simplified)
	event.ID = g.generateEventID(event)
	event.Sig = g.generateSignature(event)

	// Calculate quality score using QC system
	event.QualityScore = g.calculateQualityScore(event)

	return event
}

// generateContentForKind generates content based on persona and kind
func (g *Generator) generateContentForKind(persona Persona, kind int) string {
	// Get kind configuration
	if g.kindConfigLoader != nil {
		kindConfig, err := g.kindConfigLoader.GetKindConfig(kind)
		if err == nil {
			// Use kind-specific content length if available
			if kindConfig.ContentValidation.MaxLength > 0 {
				maxLength := kindConfig.ContentValidation.MaxLength
				if kindConfig.ContentValidation.MinLength > 0 {
					// Use a length between min and max
					length := g.rand.Intn(maxLength-kindConfig.ContentValidation.MinLength) + kindConfig.ContentValidation.MinLength
					return g.generateContentWithLength(persona, length)
				}
				return g.generateContentWithLength(persona, maxLength)
			}
		}
	}

	// Generate content based on kind requirements
	switch kind {
	case 0: // Metadata
		return g.generateMetadataContent(persona)
	case 1: // Text note
		return g.generateTextNoteContent(persona)
	case 3: // Contacts
		return g.generateContactsContent(persona)
	case 6: // Reaction
		return g.generateReactionContent(persona)
	case 7: // Repost
		return g.generateRepostContent(persona)
	case 30023: // Article
		return g.generateArticleContent(persona)
	default:
		return g.generateContent(persona)
	}
}

func (g *Generator) generateContent(persona Persona) string {
	length := g.rand.Intn(persona.Behavior.ContentLength[1]-persona.Behavior.ContentLength[0]) + persona.Behavior.ContentLength[0]
	return g.generateContentWithLength(persona, length)
}

func (g *Generator) generateContentWithLength(persona Persona, length int) string {
	// Generate content based on topics
	topic := persona.Topics[g.rand.Intn(len(persona.Topics))]

	content := fmt.Sprintf("This is a %s post about %s. ",
		persona.Name, topic)

	// Add more content to reach desired length
	for len(content) < length {
		content += "Lorem ipsum dolor sit amet, consectetur adipiscing elit. "
	}

	// Truncate to desired length
	if len(content) > length {
		content = content[:length]
	}

	return content
}

func (g *Generator) generateMetadataContent(persona Persona) string {
	// Generate profile metadata
	return fmt.Sprintf(`{"name":"%s","about":"%s user","picture":"https://example.com/avatar.jpg","banner":"https://example.com/banner.jpg"}`,
		persona.Name, persona.Name)
}

func (g *Generator) generateTextNoteContent(persona Persona) string {
	length := g.rand.Intn(persona.Behavior.ContentLength[1]-persona.Behavior.ContentLength[0]) + persona.Behavior.ContentLength[0]
	return g.generateContentWithLength(persona, length)
}

func (g *Generator) generateContactsContent(persona Persona) string {
	// Generate contacts list
	contacts := []string{}
	for i := 0; i < g.rand.Intn(5)+1; i++ {
		key, _ := btcec.NewPrivateKey()
		pubkey := fmt.Sprintf("npub1%x", key.PubKey().SerializeCompressed())
		contacts = append(contacts, pubkey)
	}

	// Convert to JSON-like format
	return fmt.Sprintf(`{"contacts":[%s]}`, strings.Join(contacts, ","))
}

func (g *Generator) generateReactionContent(persona Persona) string {
	reactions := []string{"👍", "❤️", "😂", "😢", "😡", "🤔", "🔥", "💯"}
	return reactions[g.rand.Intn(len(reactions))]
}

func (g *Generator) generateRepostContent(persona Persona) string {
	// Generate a repost with quoted content
	originalContent := g.generateContentWithLength(persona, 100)
	return fmt.Sprintf("Reposting: %s", originalContent)
}

func (g *Generator) generateArticleContent(persona Persona) string {
	// Generate article content
	title := fmt.Sprintf("Article by %s", persona.Name)
	summary := g.generateContentWithLength(persona, 200)
	content := g.generateContentWithLength(persona, 1000)

	return fmt.Sprintf(`{"title":"%s","summary":"%s","content":"%s"}`, title, summary, content)
}

// calculateQualityScore calculates quality score using QC system
func (g *Generator) calculateQualityScore(event *models.Event) float64 {
	// Use QC system to calculate quality score if available
	if g.kindConfigLoader != nil {
		// Convert nostr.Tags to [][]string for the quality controller
		tags := make([][]string, len(event.Tags))
		for i, tag := range event.Tags {
			tags[i] = make([]string, len(tag))
			copy(tags[i], tag)
		}

		score, err := g.kindConfigLoader.CalculateQualityScore(event.Kind, event.Content, tags)
		if err == nil {
			return score
		}
	}

	// Fallback to basic quality calculation
	return event.CalculateQualityScore()
}

func (g *Generator) generateTags(persona Persona, kind int) nostr.Tags {
	var tags nostr.Tags

	// Get kind configuration for this event kind
	if g.kindConfigLoader != nil {
		kindConfig, err := g.kindConfigLoader.GetKindConfig(kind)
		if err == nil {
			// Generate required tags based on kind configuration
			for _, requiredTag := range kindConfig.RequiredTags {
				tag := g.generateTagForName(requiredTag, persona)
				if tag != nil {
					tags = append(tags, *tag)
				}
			}

			// Add optional tags based on persona behavior
			optionalTagCount := g.rand.Intn(persona.Behavior.TagCount[1]-persona.Behavior.TagCount[0]) + persona.Behavior.TagCount[0]
			for i := 0; i < optionalTagCount; i++ {
				// Select a random optional tag
				if len(kindConfig.OptionalTags) > 0 {
					tagName := kindConfig.OptionalTags[g.rand.Intn(len(kindConfig.OptionalTags))]
					tag := g.generateTagForName(tagName, persona)
					if tag != nil {
						tags = append(tags, *tag)
					}
				}
			}
		} else {
			// Fallback to basic tag generation
			return g.generateBasicTags(persona)
		}
	} else {
		// Fallback to basic tag generation
		return g.generateBasicTags(persona)
	}

	return tags
}

// generateTagForName generates a tag based on tag name
func (g *Generator) generateTagForName(tagName string, persona Persona) *nostr.Tag {
	switch tagName {
	case "p":
		// Generate a pubkey tag
		key, _ := btcec.NewPrivateKey()
		pubkey := fmt.Sprintf("npub1%x", key.PubKey().SerializeCompressed())
		return &nostr.Tag{tagName, pubkey}
	case "e":
		// Generate an event tag
		eventID := fmt.Sprintf("event_%d", g.rand.Intn(10000))
		return &nostr.Tag{tagName, eventID}
	case "t":
		// Generate a hashtag
		hashtag := persona.Topics[g.rand.Intn(len(persona.Topics))]
		return &nostr.Tag{tagName, hashtag}
	case "a":
		// Generate an a tag (kind:pubkey:d-tag)
		key, _ := btcec.NewPrivateKey()
		pubkey := fmt.Sprintf("npub1%x", key.PubKey().SerializeCompressed())
		dTag := fmt.Sprintf("d_tag_%d", g.rand.Intn(1000))
		return &nostr.Tag{tagName, fmt.Sprintf("%d:%s:%s", g.rand.Intn(10)+1, pubkey, dTag)}
	case "d":
		// Generate a d tag
		dTag := fmt.Sprintf("d_tag_%d", g.rand.Intn(1000))
		return &nostr.Tag{tagName, dTag}
	case "title":
		// Generate a title tag
		title := fmt.Sprintf("Title %d", g.rand.Intn(1000))
		return &nostr.Tag{tagName, title}
	case "summary":
		// Generate a summary tag
		summary := fmt.Sprintf("Summary %d", g.rand.Intn(1000))
		return &nostr.Tag{tagName, summary}
	case "image":
		// Generate an image tag
		image := fmt.Sprintf("https://example.com/image%d.jpg", g.rand.Intn(1000))
		return &nostr.Tag{tagName, image}
	case "published_at":
		// Generate a published_at tag
		timestamp := time.Now().Add(-time.Duration(g.rand.Intn(86400)) * time.Second).Unix()
		return &nostr.Tag{tagName, fmt.Sprintf("%d", timestamp)}
	default:
		// Generic tag
		value := fmt.Sprintf("value_%d", g.rand.Intn(1000))
		return &nostr.Tag{tagName, value}
	}
}

// generateBasicTags generates basic tags when no kind config is available
func (g *Generator) generateBasicTags(persona Persona) nostr.Tags {
	tagCount := g.rand.Intn(persona.Behavior.TagCount[1]-persona.Behavior.TagCount[0]) + persona.Behavior.TagCount[0]

	var tags nostr.Tags
	for i := 0; i < tagCount; i++ {
		tagType := []string{"t", "p", "e", "a"}[g.rand.Intn(4)]
		tagValue := fmt.Sprintf("tag_value_%d", i)
		tags = append(tags, nostr.Tag{tagType, tagValue})
	}

	return tags
}

func (g *Generator) selectKind(persona Persona) int {
	if len(g.availableKinds) == 0 {
		return 1 // fallback to text note
	}

	// Weighted selection based on persona behavior and available kinds
	var kinds []int
	var weights []float64

	// Define persona-specific kind preferences
	switch persona.Name {
	case "Spammer":
		// Spammers prefer text notes and reactions
		kinds = []int{1, 6, 7} // text note, reaction, repost
		weights = []float64{0.8, 0.15, 0.05}
	case "Influencer":
		// Influencers use more diverse kinds
		kinds = []int{1, 0, 3, 6, 7, 30023} // text note, metadata, contacts, reaction, repost, article
		weights = []float64{0.4, 0.1, 0.1, 0.2, 0.1, 0.1}
	case "Casual":
		// Casual users mostly text notes with some reactions
		kinds = []int{1, 6, 7}
		weights = []float64{0.7, 0.2, 0.1}
	default:
		// Default: mostly text notes
		kinds = []int{1, 6, 7}
		weights = []float64{0.8, 0.1, 0.1}
	}

	// Filter to only available kinds
	var availableKinds []int
	var availableWeights []float64
	totalWeight := 0.0

	for i, kind := range kinds {
		if g.isKindAvailable(kind) {
			availableKinds = append(availableKinds, kind)
			availableWeights = append(availableWeights, weights[i])
			totalWeight += weights[i]
		}
	}

	// Normalize weights
	for i := range availableWeights {
		availableWeights[i] /= totalWeight
	}

	if len(availableKinds) == 0 {
		return g.availableKinds[g.rand.Intn(len(g.availableKinds))]
	}

	// Select based on weights
	rand := g.rand.Float64()
	cumulative := 0.0

	for i, weight := range availableWeights {
		cumulative += weight
		if rand <= cumulative {
			return availableKinds[i]
		}
	}

	return availableKinds[0] // fallback
}

// isKindAvailable checks if a kind is available in our QC system
func (g *Generator) isKindAvailable(kind int) bool {
	for _, availableKind := range g.availableKinds {
		if availableKind == kind {
			return true
		}
	}
	return false
}

func (g *Generator) generateEventID(event *models.Event) string {
	// Simplified event ID generation
	data := fmt.Sprintf("%s%s%d%s", event.PubKey, event.CreatedAt.Time().Format(time.RFC3339), event.Kind, event.Content)
	hash := chainhash.HashH([]byte(data))
	return hash.String()
}

func (g *Generator) generateSignature(event *models.Event) string {
	// Simplified signature generation
	// In a real implementation, this would use the private key
	return fmt.Sprintf("sig_%s", event.ID[:16])
}

func (g *Generator) ExportEvents(events []*models.Event, writer io.Writer, format string) error {
	switch format {
	case "json":
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(events)
	case "nostr":
		for _, event := range events {
			nostrEvent := event.ToNostrEvent()
			if err := json.NewEncoder(writer).Encode(nostrEvent); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

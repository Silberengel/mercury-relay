package testgen

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"time"

	"mercury-relay/internal/config"
	"mercury-relay/internal/models"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/nbd-wtf/go-nostr"
)

type Generator struct {
	config *config.Config
	rand   *rand.Rand
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
	return &Generator{
		config: config,
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
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
	// Generate content based on persona behavior
	content := g.generateContent(persona)

	// Generate tags
	tags := g.generateTags(persona)

	// Create event
	event := &models.Event{
		PubKey:      persona.PubKey,
		CreatedAt:   time.Now().Add(-time.Duration(g.rand.Intn(3600)) * time.Second), // Random time in last hour
		Kind:        g.selectKind(persona),
		Tags:        tags,
		Content:     content,
		CreatedAtDB: time.Now(),
	}

	// Generate ID and signature (simplified)
	event.ID = g.generateEventID(event)
	event.Sig = g.generateSignature(event)

	// Calculate quality score
	event.QualityScore = event.CalculateQualityScore()

	return event
}

func (g *Generator) generateContent(persona Persona) string {
	length := g.rand.Intn(persona.Behavior.ContentLength[1]-persona.Behavior.ContentLength[0]) + persona.Behavior.ContentLength[0]

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

func (g *Generator) generateTags(persona Persona) nostr.Tags {
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
	// Weighted selection based on persona behavior
	kinds := []int{1, 6, 7}             // text note, reaction, repost
	weights := []float64{0.8, 0.1, 0.1} // mostly text notes

	if persona.Name == "Spammer" {
		weights = []float64{0.9, 0.05, 0.05} // even more text notes
	}

	rand := g.rand.Float64()
	cumulative := 0.0

	for i, weight := range weights {
		cumulative += weight
		if rand <= cumulative {
			return kinds[i]
		}
	}

	return 1 // default to text note
}

func (g *Generator) generateEventID(event *models.Event) string {
	// Simplified event ID generation
	data := fmt.Sprintf("%s%s%d%s", event.PubKey, event.CreatedAt.Format(time.RFC3339), event.Kind, event.Content)
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

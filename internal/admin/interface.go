package admin

import (
	"fmt"
	"log"

	"mercury-relay/internal/config"
	"mercury-relay/internal/quality"
)

type Interface struct {
	config         *config.Config
	qualityControl *quality.Controller
}

func NewInterface(config *config.Config) *Interface {
	return &Interface{
		config: config,
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
	// This would start the terminal user interface
	// For now, just log that it's starting
	log.Println("Starting admin TUI interface...")
	return fmt.Errorf("TUI not implemented yet")
}

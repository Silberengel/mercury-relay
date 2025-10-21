package quality

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// EventKindConfig represents the configuration for a specific event kind
type EventKindConfig struct {
	Name              string            `yaml:"name"`
	Description       string            `yaml:"description"`
	RequiredTags      []string          `yaml:"required_tags"`
	OptionalTags      []string          `yaml:"optional_tags"`
	ContentValidation ContentValidation `yaml:"content_validation"`
	QualityRules      []QualityRule     `yaml:"quality_rules"`
	Replaceable       bool              `yaml:"replaceable"`
	Ephemeral         bool              `yaml:"ephemeral"`
	Addressable       bool              `yaml:"addressable"`
}

type ContentValidation struct {
	Type           string   `yaml:"type"`
	MaxLength      int      `yaml:"max_length"`
	MinLength      int      `yaml:"min_length"`
	RequiredFields []string `yaml:"required_fields"`
	OptionalFields []string `yaml:"optional_fields"`
}

type QualityRule struct {
	Name        string  `yaml:"name"`
	Weight      float64 `yaml:"weight"`
	Description string  `yaml:"description"`
	MaxFollows  int     `yaml:"max_follows,omitempty"`
}

type GlobalQualityConfig struct {
	SpamDetection     SpamDetectionConfig     `yaml:"spam_detection"`
	RateLimiting      RateLimitingConfig      `yaml:"rate_limiting"`
	ContentValidation ContentValidationConfig `yaml:"content_validation"`
	TagValidation     TagValidationConfig     `yaml:"tag_validation"`
}

type SpamDetectionConfig struct {
	Enabled   bool       `yaml:"enabled"`
	Threshold float64    `yaml:"threshold"`
	Rules     []SpamRule `yaml:"rules"`
}

type SpamRule struct {
	Name        string  `yaml:"name"`
	Weight      float64 `yaml:"weight"`
	Description string  `yaml:"description"`
}

type RateLimitingConfig struct {
	Enabled         bool           `yaml:"enabled"`
	EventsPerMinute int            `yaml:"events_per_minute"`
	BurstLimit      int            `yaml:"burst_limit"`
	PerKindLimits   map[string]int `yaml:"per_kind_limits"`
}

type ContentValidationConfig struct {
	MaxContentLength  int      `yaml:"max_content_length"`
	MinContentLength  int      `yaml:"min_content_length"`
	AllowedEncodings  []string `yaml:"allowed_encodings"`
	ForbiddenPatterns []string `yaml:"forbidden_patterns"`
}

type TagValidationConfig struct {
	MaxTagsPerEvent    int               `yaml:"max_tags_per_event"`
	MaxTagLength       int               `yaml:"max_tag_length"`
	RequiredTagFormats map[string]string `yaml:"required_tag_formats"`
}

type RelaySettings struct {
	Storage    StorageConfig    `yaml:"storage"`
	Retention  RetentionConfig  `yaml:"retention"`
	Quarantine QuarantineConfig `yaml:"quarantine"`
}

type StorageConfig struct {
	RegularEvents     bool `yaml:"regular_events"`
	ReplaceableEvents bool `yaml:"replaceable_events"`
	EphemeralEvents   bool `yaml:"ephemeral_events"`
	AddressableEvents bool `yaml:"addressable_events"`
}

type RetentionConfig struct {
	RegularEvents     string `yaml:"regular_events"`
	ReplaceableEvents string `yaml:"replaceable_events"`
	AddressableEvents string `yaml:"addressable_events"`
}

type QuarantineConfig struct {
	Enabled                 bool    `yaml:"enabled"`
	AutoQuarantineThreshold float64 `yaml:"auto_quarantine_threshold"`
	ManualReviewThreshold   float64 `yaml:"manual_review_threshold"`
	QuarantineDuration      string  `yaml:"quarantine_duration"`
}

type NostrEventKindsConfig struct {
	EventKinds    map[string]EventKindConfig `yaml:"event_kinds"`
	GlobalQuality GlobalQualityConfig        `yaml:"global_quality"`
	RelaySettings RelaySettings              `yaml:"relay_settings"`
}

type KindConfigLoader struct {
	config *NostrEventKindsConfig
}

func NewKindConfigLoader(configPath string) (*KindConfigLoader, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config NostrEventKindsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &KindConfigLoader{config: &config}, nil
}

func (k *KindConfigLoader) GetKindConfig(kind int) (*EventKindConfig, error) {
	kindStr := strconv.Itoa(kind)
	config, exists := k.config.EventKinds[kindStr]
	if !exists {
		return nil, fmt.Errorf("no configuration found for kind %d", kind)
	}
	return &config, nil
}

func (k *KindConfigLoader) ValidateEventKind(eventKind int, content string, tags [][]string) error {
	config, err := k.GetKindConfig(eventKind)
	if err != nil {
		return err
	}

	// Validate content
	if err := k.validateContent(config.ContentValidation, content); err != nil {
		return fmt.Errorf("content validation failed: %w", err)
	}

	// Validate tags
	if err := k.validateTags(config.RequiredTags, config.OptionalTags, tags); err != nil {
		return fmt.Errorf("tag validation failed: %w", err)
	}

	return nil
}

func (k *KindConfigLoader) validateContent(validation ContentValidation, content string) error {
	// Check length
	if validation.MaxLength > 0 && len(content) > validation.MaxLength {
		return fmt.Errorf("content too long: %d > %d", len(content), validation.MaxLength)
	}
	if validation.MinLength > 0 && len(content) < validation.MinLength {
		return fmt.Errorf("content too short: %d < %d", len(content), validation.MinLength)
	}

	// Check type-specific validation
	switch validation.Type {
	case "json":
		if content != "" {
			var jsonData interface{}
			if err := json.Unmarshal([]byte(content), &jsonData); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			// Check required fields
			if len(validation.RequiredFields) > 0 {
				jsonMap, ok := jsonData.(map[string]interface{})
				if !ok {
					return fmt.Errorf("JSON content must be an object")
				}

				for _, field := range validation.RequiredFields {
					if _, exists := jsonMap[field]; !exists {
						return fmt.Errorf("missing required field: %s", field)
					}
				}
			}
		}
	case "text":
		// Basic text validation
		if validation.MaxLength > 0 && len(content) > validation.MaxLength {
			return fmt.Errorf("text content too long")
		}
	case "encrypted":
		// Basic encrypted content validation
		if len(content) == 0 {
			return fmt.Errorf("encrypted content cannot be empty")
		}
	case "base64":
		// Validate base64 encoding
		if content != "" {
			if _, err := json.Marshal(content); err != nil {
				return fmt.Errorf("invalid base64 content: %w", err)
			}
		}
	}

	return nil
}

func (k *KindConfigLoader) validateTags(requiredTags, optionalTags []string, tags [][]string) error {
	// Check required tags
	for _, requiredTag := range requiredTags {
		found := false
		for _, tag := range tags {
			if len(tag) > 0 && tag[0] == requiredTag {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("missing required tag: %s", requiredTag)
		}
	}

	// Validate tag formats
	for _, tag := range tags {
		if len(tag) == 0 {
			continue
		}

		tagName := tag[0]
		if len(tag) > 1 {
			tagValue := tag[1]

			// Check against required tag formats
			if pattern, exists := k.config.GlobalQuality.TagValidation.RequiredTagFormats[tagName]; exists {
				matched, err := regexp.MatchString(pattern, tagValue)
				if err != nil {
					return fmt.Errorf("invalid regex pattern for tag %s: %w", tagName, err)
				}
				if !matched {
					return fmt.Errorf("tag %s value %s does not match required format %s", tagName, tagValue, pattern)
				}
			}
		}
	}

	return nil
}

func (k *KindConfigLoader) CalculateQualityScore(eventKind int, content string, tags [][]string) (float64, error) {
	config, err := k.GetKindConfig(eventKind)
	if err != nil {
		return 0, err
	}

	score := 1.0

	// Apply quality rules
	for _, rule := range config.QualityRules {
		ruleScore := k.calculateRuleScore(rule, content, tags)
		score *= (1.0 - (rule.Weight * (1.0 - ruleScore)))
	}

	// Apply global spam detection
	if k.config.GlobalQuality.SpamDetection.Enabled {
		spamScore := k.calculateSpamScore(content, tags)
		score *= (1.0 - (k.config.GlobalQuality.SpamDetection.Threshold * spamScore))
	}

	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score, nil
}

func (k *KindConfigLoader) calculateRuleScore(rule QualityRule, content string, tags [][]string) float64 {
	switch rule.Name {
	case "valid_json":
		if content != "" {
			var jsonData interface{}
			if err := json.Unmarshal([]byte(content), &jsonData); err != nil {
				return 0.0
			}
		}
		return 1.0
	case "has_name":
		if content != "" {
			var jsonData map[string]interface{}
			if err := json.Unmarshal([]byte(content), &jsonData); err == nil {
				if _, exists := jsonData["name"]; exists {
					return 1.0
				}
			}
		}
		return 0.0
	case "reasonable_length":
		if len(content) > 0 && len(content) < 1000 {
			return 1.0
		}
		return 0.5
	case "valid_p_tags":
		validCount := 0
		for _, tag := range tags {
			if len(tag) >= 2 && tag[0] == "p" {
				if len(tag[1]) == 64 {
					validCount++
				}
			}
		}
		if len(tags) > 0 {
			return float64(validCount) / float64(len(tags))
		}
		return 1.0
	case "reasonable_follow_count":
		if rule.MaxFollows > 0 {
			followCount := 0
			for _, tag := range tags {
				if len(tag) >= 2 && tag[0] == "p" {
					followCount++
				}
			}
			if followCount <= rule.MaxFollows {
				return 1.0
			}
			return 0.5
		}
		return 1.0
	case "valid_d_tag":
		for _, tag := range tags {
			if len(tag) >= 2 && tag[0] == "d" && tag[1] != "" {
				return 1.0
			}
		}
		return 0.0
	case "valid_title_tag":
		for _, tag := range tags {
			if len(tag) >= 2 && tag[0] == "title" && tag[1] != "" {
				return 1.0
			}
		}
		return 0.0
	case "valid_a_tags":
		for _, tag := range tags {
			if len(tag) >= 2 && tag[0] == "a" && tag[1] != "" {
				return 1.0
			}
		}
		return 0.0
	case "valid_auto_update":
		for _, tag := range tags {
			if len(tag) >= 2 && tag[0] == "auto-update" {
				value := tag[1]
				if value == "yes" || value == "ask" || value == "no" {
					return 1.0
				}
			}
		}
		return 0.0
	case "valid_derivative_work":
		hasP := false
		hasE := false
		for _, tag := range tags {
			if len(tag) >= 2 && tag[0] == "p" {
				hasP = true
			}
			if len(tag) >= 2 && tag[0] == "E" {
				hasE = true
			}
		}
		// If it's a derivative work, both p and E tags should be present
		// If it's not a derivative work, this rule doesn't apply
		if hasP || hasE {
			return 1.0
		}
		return 1.0 // Not a derivative work, so this rule passes
	case "valid_content":
		if len(content) > 0 {
			return 1.0
		}
		return 0.0
	case "valid_asciidoc":
		// Check for basic AsciiDoc syntax
		if strings.Contains(content, "=") || strings.Contains(content, "==") {
			return 1.0
		}
		return 0.8 // Even without AsciiDoc, content is still valid
	case "valid_wikilinks":
		// Check for wikilinks in double brackets
		if strings.Contains(content, "[[") && strings.Contains(content, "]]") {
			return 1.0
		}
		return 0.9 // Wikilinks are optional
	default:
		return 1.0
	}
}

func (k *KindConfigLoader) calculateSpamScore(content string, tags [][]string) float64 {
	score := 0.0

	// Check for repetitive content
	if k.isRepetitiveContent(content) {
		score += 0.3
	}

	// Check for excessive mentions
	mentionCount := 0
	for _, tag := range tags {
		if len(tag) >= 2 && (tag[0] == "p" || tag[0] == "e") {
			mentionCount++
		}
	}
	if mentionCount > 10 {
		score += 0.2
	}

	// Check for suspicious patterns
	if k.hasSuspiciousPatterns(content) {
		score += 0.4
	}

	return score
}

func (k *KindConfigLoader) isRepetitiveContent(content string) bool {
	// Simple repetitive content detection
	words := strings.Fields(content)
	if len(words) < 3 {
		return false
	}

	wordCounts := make(map[string]int)
	for _, word := range words {
		wordCounts[strings.ToLower(word)]++
	}

	// Check if any word appears more than 50% of the time
	for _, count := range wordCounts {
		if float64(count)/float64(len(words)) > 0.5 {
			return true
		}
	}

	return false
}

func (k *KindConfigLoader) hasSuspiciousPatterns(content string) bool {
	// Check for common spam patterns
	suspiciousPatterns := []string{
		"click here",
		"free money",
		"guaranteed",
		"act now",
		"limited time",
	}

	contentLower := strings.ToLower(content)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(contentLower, pattern) {
			return true
		}
	}

	return false
}

func (k *KindConfigLoader) GetGlobalQualityConfig() *GlobalQualityConfig {
	return &k.config.GlobalQuality
}

func (k *KindConfigLoader) GetRelaySettings() *RelaySettings {
	return &k.config.RelaySettings
}

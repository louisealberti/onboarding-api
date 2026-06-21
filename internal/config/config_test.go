package config

import "testing"

func TestWebhookEnabled_URLSet(t *testing.T) {
	c := &Config{WebhookURL: "https://example.com/hooks"}
	if !c.WebhookEnabled() {
		t.Error("expected WebhookEnabled to be true when WebhookURL is set")
	}
}

func TestWebhookEnabled_URLEmpty(t *testing.T) {
	c := &Config{}
	if c.WebhookEnabled() {
		t.Error("expected WebhookEnabled to be false when WebhookURL is empty")
	}
}

func baseValidConfig() *Config {
	return &Config{
		DBHost: "localhost", DBUser: "u", DBPassword: "p", DBName: "db",
	}
}

func TestValidate_AcceptsValidHTTPSWebhookURL(t *testing.T) {
	c := baseValidConfig()
	c.WebhookURL = "https://example.com/hooks/customers"
	if err := c.validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidate_AcceptsValidHTTPWebhookURL(t *testing.T) {
	c := baseValidConfig()
	c.WebhookURL = "http://localhost:9000/hooks"
	if err := c.validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidate_AcceptsEmptyWebhookURL(t *testing.T) {
	c := baseValidConfig()
	if err := c.validate(); err != nil {
		t.Errorf("expected no error when WebhookURL is empty, got %v", err)
	}
}

func TestValidate_RejectsMalformedWebhookURL(t *testing.T) {
	c := baseValidConfig()
	c.WebhookURL = "not a url"
	if err := c.validate(); err == nil {
		t.Error("expected validation error for malformed WEBHOOK_URL")
	}
}

func TestValidate_RejectsWebhookURLWithoutScheme(t *testing.T) {
	c := baseValidConfig()
	c.WebhookURL = "example.com/hooks"
	if err := c.validate(); err == nil {
		t.Error("expected validation error for WEBHOOK_URL without http(s) scheme")
	}
}

func TestValidate_RejectsNonHTTPScheme(t *testing.T) {
	c := baseValidConfig()
	c.WebhookURL = "ftp://example.com/hooks"
	if err := c.validate(); err == nil {
		t.Error("expected validation error for non-http(s) WEBHOOK_URL scheme")
	}
}

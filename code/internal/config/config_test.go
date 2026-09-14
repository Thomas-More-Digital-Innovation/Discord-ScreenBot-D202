package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveOutput(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		input       string
		expectedID  int
		expectError bool
	}{
		{"screena", 1, false},
		{"ScreenA", 1, false},
		{"screen-a", 1, false},
		{"screen_b", 2, false},
		{"1", 1, false},
		{"output3", 3, false},
		{"out4", 4, false},
		{"", 0, true},
		{"nonexistent_screen_xyz", 0, true},
	}

	for _, tc := range tests {
		id, _, err := cfg.ResolveOutput(tc.input)
		if tc.expectError && err == nil {
			t.Errorf("ResolveOutput(%q) expected error, got nil", tc.input)
		}
		if !tc.expectError && err != nil {
			t.Errorf("ResolveOutput(%q) unexpected error: %v", tc.input, err)
		}
		if !tc.expectError && id != tc.expectedID {
			t.Errorf("ResolveOutput(%q) expected %d, got %d", tc.input, tc.expectedID, id)
		}
	}
}

func TestResolveInput(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		input       string
		expectedID  int
		expectError bool
	}{
		{"input1", 1, false},
		{"input2", 2, false},
		{"Input2", 2, false},
		{"input-2", 2, false},
		{"4", 4, false},
		{"in5", 5, false},
		{"", 0, true},
		{"unknown_device_999", 0, true},
	}

	for _, tc := range tests {
		id, _, err := cfg.ResolveInput(tc.input)
		if tc.expectError && err == nil {
			t.Errorf("ResolveInput(%q) expected error, got nil", tc.input)
		}
		if !tc.expectError && err != nil {
			t.Errorf("ResolveInput(%q) unexpected error: %v", tc.input, err)
		}
		if !tc.expectError && id != tc.expectedID {
			t.Errorf("ResolveInput(%q) expected %d, got %d", tc.input, tc.expectedID, id)
		}
	}
}

func TestLoadConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `
discord:
  token: "test-token"
  guild_id: "123456"
  channel_id: "987654321"

amx:
  host: "http://10.0.0.1"
  timeout: "10s"

mapping:
  outputs:
    projector: 3
  inputs:
    laptop: 1
    camera: 2
`
	if err := os.WriteFile(configFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Discord.Token != "test-token" {
		t.Errorf("expected token 'test-token', got %q", cfg.Discord.Token)
	}
	if cfg.Discord.GuildID != "123456" {
		t.Errorf("expected guild_id '123456', got %q", cfg.Discord.GuildID)
	}
	if cfg.Discord.ChannelID != "987654321" {
		t.Errorf("expected channel_id '987654321', got %q", cfg.Discord.ChannelID)
	}
	if cfg.AMX.Host != "http://10.0.0.1" {
		t.Errorf("expected host 'http://10.0.0.1', got %q", cfg.AMX.Host)
	}

	id, _, err := cfg.ResolveOutput("projector")
	if err != nil || id != 3 {
		t.Errorf("expected projector -> 3, got id=%d err=%v", id, err)
	}

	id, _, err = cfg.ResolveInput("laptop")
	if err != nil || id != 1 {
		t.Errorf("expected laptop -> 1, got id=%d err=%v", id, err)
	}
}

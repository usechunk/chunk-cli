package sources

import (
	"testing"
)

func TestNewSourceManager(t *testing.T) {
	manager := NewSourceManager()

	if manager == nil {
		t.Fatal("Expected NewSourceManager to return non-nil manager")
	}

	if manager.chunkhub == nil {
		t.Error("Expected chunkhub client to be initialized")
	}

	if manager.github == nil {
		t.Error("Expected github client to be initialized")
	}

	if manager.modrinth == nil {
		t.Error("Expected modrinth client to be initialized")
	}

	if manager.local == nil {
		t.Error("Expected local client to be initialized")
	}

	if manager.recipe == nil {
		t.Error("Expected recipe client to be initialized")
	}
}

func TestGetClient(t *testing.T) {
	manager := NewSourceManager()

	tests := []struct {
		name       string
		sourceType string
		wantErr    bool
	}{
		{
			name:       "recipe client",
			sourceType: "recipe",
			wantErr:    false,
		},
		{
			name:       "chunkhub client",
			sourceType: "chunkhub",
			wantErr:    false,
		},
		{
			name:       "github client",
			sourceType: "github",
			wantErr:    false,
		},
		{
			name:       "modrinth client",
			sourceType: "modrinth",
			wantErr:    false,
		},
		{
			name:       "local client",
			sourceType: "local",
			wantErr:    false,
		},
		{
			name:       "unknown client",
			sourceType: "unknown",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := manager.GetClient(tt.sourceType)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetClient(%q) error = %v, wantErr %v", tt.sourceType, err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Errorf("GetClient(%q) returned nil client", tt.sourceType)
			}
		})
	}
}

package metadata

import (
	"testing"
	"time"
)

func TestCacheEntry_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "expired entry",
			expiresAt: time.Now().Add(-1 * time.Hour),
			want:      true,
		},
		{
			name:      "valid entry",
			expiresAt: time.Now().Add(1 * time.Hour),
			want:      false,
		},
		{
			name:      "just expired",
			expiresAt: time.Now().Add(-1 * time.Millisecond),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CacheEntry{
				ExpiresAt: tt.expiresAt,
			}
			if got := c.IsExpired(); got != tt.want {
				t.Errorf("CacheEntry.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoaderType_Constants(t *testing.T) {
	tests := []struct {
		loader   LoaderType
		expected string
	}{
		{LoaderForge, "forge"},
		{LoaderFabric, "fabric"},
		{LoaderQuilt, "quilt"},
		{LoaderNeoForge, "neoforge"},
	}

	for _, tt := range tests {
		t.Run(string(tt.loader), func(t *testing.T) {
			if string(tt.loader) != tt.expected {
				t.Errorf("LoaderType = %v, want %v", tt.loader, tt.expected)
			}
		})
	}
}

func TestVersionType_Constants(t *testing.T) {
	tests := []struct {
		vtype    VersionType
		expected string
	}{
		{VersionRelease, "release"},
		{VersionSnapshot, "snapshot"},
		{VersionBeta, "old_beta"},
		{VersionAlpha, "old_alpha"},
	}

	for _, tt := range tests {
		t.Run(string(tt.vtype), func(t *testing.T) {
			if string(tt.vtype) != tt.expected {
				t.Errorf("VersionType = %v, want %v", tt.vtype, tt.expected)
			}
		})
	}
}

func TestJavaVersionRequirements(t *testing.T) {
	tests := []struct {
		mcVersion   string
		minJava     int
		description string
	}{
		{"1.21", 21, "MC 1.21 requires Java 21"},
		{"1.20.5", 21, "MC 1.20.5 requires Java 21"},
		{"1.20", 17, "MC 1.20 requires Java 17"},
		{"1.19", 17, "MC 1.19 requires Java 17"},
		{"1.18", 17, "MC 1.18 requires Java 17"},
		{"1.17", 16, "MC 1.17 requires Java 16"},
		{"1.16", 8, "MC 1.16 requires Java 8"},
		{"1.12", 8, "MC 1.12 requires Java 8"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			if got := JavaVersionRequirements[tt.mcVersion]; got != tt.minJava {
				t.Errorf("JavaVersionRequirements[%s] = %v, want %v", tt.mcVersion, got, tt.minJava)
			}
		})
	}
}

func TestMinecraftVersion_Struct(t *testing.T) {
	now := time.Now()
	v := MinecraftVersion{
		ID:          "1.20.4",
		Type:        VersionRelease,
		URL:         "https://example.com/version.json",
		Time:        now,
		ReleaseTime: now,
		JavaVersion: 17,
	}

	if v.ID != "1.20.4" {
		t.Errorf("MinecraftVersion.ID = %v, want 1.20.4", v.ID)
	}
	if v.Type != VersionRelease {
		t.Errorf("MinecraftVersion.Type = %v, want release", v.Type)
	}
	if v.JavaVersion != 17 {
		t.Errorf("MinecraftVersion.JavaVersion = %v, want 17", v.JavaVersion)
	}
}

func TestLoaderVersion_Struct(t *testing.T) {
	now := time.Now()
	v := LoaderVersion{
		Version:          "0.14.21",
		MinecraftVersion: "1.20.4",
		Stable:           true,
		LoaderType:       LoaderFabric,
		ReleaseDate:      now,
	}

	if v.Version != "0.14.21" {
		t.Errorf("LoaderVersion.Version = %v, want 0.14.21", v.Version)
	}
	if v.MinecraftVersion != "1.20.4" {
		t.Errorf("LoaderVersion.MinecraftVersion = %v, want 1.20.4", v.MinecraftVersion)
	}
	if !v.Stable {
		t.Error("LoaderVersion.Stable should be true")
	}
	if v.LoaderType != LoaderFabric {
		t.Errorf("LoaderVersion.LoaderType = %v, want fabric", v.LoaderType)
	}
}

func TestForgeVersion_Struct(t *testing.T) {
	v := ForgeVersion{
		Version:          "47.2.0",
		MinecraftVersion: "1.20.1",
		IsRecommended:    true,
		IsLatest:         false,
	}

	if v.Version != "47.2.0" {
		t.Errorf("ForgeVersion.Version = %v, want 47.2.0", v.Version)
	}
	if !v.IsRecommended {
		t.Error("ForgeVersion.IsRecommended should be true")
	}
}

func TestNeoForgeVersion_Struct(t *testing.T) {
	v := NeoForgeVersion{
		Version:          "21.1.1",
		MinecraftVersion: "1.21.1",
	}

	if v.Version != "21.1.1" {
		t.Errorf("NeoForgeVersion.Version = %v, want 21.1.1", v.Version)
	}
	if v.MinecraftVersion != "1.21.1" {
		t.Errorf("NeoForgeVersion.MinecraftVersion = %v, want 1.21.1", v.MinecraftVersion)
	}
}

func TestMinecraftVersionManifest_Struct(t *testing.T) {
	manifest := MinecraftVersionManifest{
		Latest: LatestVersions{
			Release:  "1.20.4",
			Snapshot: "24w05a",
		},
		Versions: []*MinecraftVersion{
			{ID: "1.20.4", Type: VersionRelease},
			{ID: "24w05a", Type: VersionSnapshot},
		},
	}

	if manifest.Latest.Release != "1.20.4" {
		t.Errorf("Latest.Release = %v, want 1.20.4", manifest.Latest.Release)
	}
	if manifest.Latest.Snapshot != "24w05a" {
		t.Errorf("Latest.Snapshot = %v, want 24w05a", manifest.Latest.Snapshot)
	}
	if len(manifest.Versions) != 2 {
		t.Errorf("len(Versions) = %v, want 2", len(manifest.Versions))
	}
}

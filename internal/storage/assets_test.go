package storage

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/zhuangkaiyi/gang-chat/server/internal/config"
)

func testS3Config() *config.Config {
	return &config.Config{
		S3Endpoint:        "https://os.ky-z.com:9000",
		S3Bucket:          "gang-chat",
		S3Region:          "us-east-1",
		S3AccessKeyID:     "sid",
		S3SecretAccessKey: "skey",
		S3ForcePathStyle:  true,
	}
}

func TestNewAssetStorageRequiresS3Config(t *testing.T) {
	cfg := testS3Config()
	cfg.AssetObjectPrefix = "room-assets"
	store, err := NewAssetStorage(cfg)
	if err != nil {
		t.Fatalf("NewAssetStorage returned error: %v", err)
	}
	if !store.RemoteEnabled() {
		t.Fatalf("S3 storage should be enabled")
	}

	key := store.ObjectKey("asset_1", "room.png")
	if key != "room-assets/asset_1/room.png" {
		t.Fatalf("unexpected object key: %q", key)
	}
	if got := store.PublicURL(key, "asset_1", "room.png"); got != "/assets/asset_1/room.png" {
		t.Fatalf("unexpected proxied asset URL: %q", got)
	}
}

func TestNewAssetStorageReportsIncompleteS3Config(t *testing.T) {
	_, err := NewAssetStorage(&config.Config{
		S3Endpoint: "https://os.ky-z.com:9000",
		S3Bucket:   "gang-chat",
	})
	if err == nil {
		t.Fatalf("expected missing S3 credentials error")
	}
	if !strings.Contains(err.Error(), "s3_access_key_id") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAssetStorageBuildsExpiringCacheHeadersFromTTL(t *testing.T) {
	cfg := testS3Config()
	cfg.AssetCacheTTLSeconds = 60
	cfg.AssetObjectPrefix = "assets"
	store, err := NewAssetStorage(cfg)
	if err != nil {
		t.Fatalf("NewAssetStorage returned error: %v", err)
	}

	headers := http.Header{}
	now := time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)
	store.ApplyCacheHeaders(headers, now)

	if got := headers.Get("Cache-Control"); got != "public, max-age=60, immutable" {
		t.Fatalf("unexpected Cache-Control: %q", got)
	}
	if got := headers.Get("Expires"); got != "Thu, 18 Jun 2026 10:01:00 GMT" {
		t.Fatalf("unexpected Expires: %q", got)
	}
}

func TestAssetStorageHonorsExplicitCacheControl(t *testing.T) {
	cfg := testS3Config()
	cfg.AssetCacheControl = "private, max-age=5"
	store, err := NewAssetStorage(cfg)
	if err != nil {
		t.Fatalf("NewAssetStorage returned error: %v", err)
	}

	headers := http.Header{}
	store.ApplyCacheHeaders(headers, time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC))

	if got := headers.Get("Cache-Control"); got != "private, max-age=5" {
		t.Fatalf("unexpected Cache-Control: %q", got)
	}
	if got := headers.Get("Expires"); got != "Thu, 18 Jun 2026 10:00:05 GMT" {
		t.Fatalf("unexpected Expires: %q", got)
	}
}

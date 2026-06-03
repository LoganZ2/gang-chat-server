package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/tencentyun/cos-go-sdk-v5"
	"github.com/zhuangkaiyi/gang-chat/server/internal/config"
)

const defaultAssetCacheControl = "public, max-age=31536000, immutable"

type AssetStorage struct {
	cacheDir     string
	objectPrefix string
	publicBase   string
	cacheControl string
	remote       remoteStore
}

type remoteStore interface {
	PutFile(ctx context.Context, key, filePath, mimeType, cacheControl string) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

func NewAssetStorage(cfg *config.Config) (*AssetStorage, error) {
	cacheDir := "assets"
	backend := "local"
	objectPrefix := "assets"
	publicBase := ""
	cacheControl := defaultAssetCacheControl
	if cfg != nil {
		if cfg.AssetDir != "" {
			cacheDir = cfg.AssetDir
		}
		if cfg.StorageBackend != "" {
			backend = strings.ToLower(strings.TrimSpace(cfg.StorageBackend))
		}
		if cfg.AssetObjectPrefix != "" {
			objectPrefix = cfg.AssetObjectPrefix
		}
		publicBase = strings.TrimRight(strings.TrimSpace(cfg.AssetPublicBaseURL), "/")
		if cfg.AssetCacheControl != "" {
			cacheControl = strings.TrimSpace(cfg.AssetCacheControl)
		}
	}

	store := &AssetStorage{
		cacheDir:     cacheDir,
		objectPrefix: cleanObjectKey(objectPrefix),
		publicBase:   publicBase,
		cacheControl: cacheControl,
	}
	switch backend {
	case "", "local", "disk":
		return store, nil
	case "cos", "tencent-cos", "tencent_cloud_cos":
		remote, err := newCOSRemote(cfg)
		if err != nil {
			return nil, err
		}
		store.remote = remote
		return store, nil
	default:
		return nil, fmt.Errorf("unsupported storage backend %q", backend)
	}
}

func (s *AssetStorage) CacheControl() string {
	if s == nil || s.cacheControl == "" {
		return defaultAssetCacheControl
	}
	return s.cacheControl
}

func (s *AssetStorage) ObjectKey(assetID, filename string) string {
	parts := make([]string, 0, 3)
	if s != nil && s.objectPrefix != "" {
		parts = append(parts, s.objectPrefix)
	}
	parts = append(parts, assetID, filename)
	return path.Join(parts...)
}

func (s *AssetStorage) PublicURL(key, assetID, filename string) string {
	if s != nil && s.publicBase != "" {
		return s.publicBase + "/" + escapeObjectKey(cleanObjectKey(key))
	}
	return "/" + path.Join("assets", assetID, filename)
}

func (s *AssetStorage) CachePath(assetID, filename string) (string, error) {
	if assetID == "" || filename == "" {
		return "", errors.New("asset id and filename are required")
	}
	if strings.Contains(assetID, "/") || strings.Contains(assetID, "\\") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return "", errors.New("asset path must not contain separators")
	}
	cacheDir := "assets"
	if s != nil && s.cacheDir != "" {
		cacheDir = s.cacheDir
	}
	return filepath.Join(cacheDir, assetID, filename), nil
}

func (s *AssetStorage) PutCachedFile(ctx context.Context, key, filePath, mimeType string) error {
	if s == nil || s.remote == nil {
		return nil
	}
	return s.remote.PutFile(ctx, cleanObjectKey(key), filePath, mimeType, s.CacheControl())
}

func (s *AssetStorage) Delete(ctx context.Context, key string) error {
	if s == nil || s.remote == nil {
		return nil
	}
	return s.remote.Delete(ctx, cleanObjectKey(key))
}

func (s *AssetStorage) EnsureCached(ctx context.Context, key, assetID, filename string) (string, error) {
	cachePath, err := s.CachePath(assetID, filename)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(cachePath); err == nil {
		return cachePath, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if s == nil || s.remote == nil {
		return "", os.ErrNotExist
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return "", err
	}
	body, err := s.remote.Get(ctx, cleanObjectKey(key))
	if err != nil {
		return "", err
	}
	defer body.Close()

	tmp, err := os.CreateTemp(filepath.Dir(cachePath), ".asset-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if _, err := io.Copy(tmp, body); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tmpPath, cachePath); err != nil {
		if _, statErr := os.Stat(cachePath); statErr == nil {
			return cachePath, nil
		}
		return "", err
	}
	return cachePath, nil
}

type cosRemote struct {
	client *cos.Client
}

func newCOSRemote(cfg *config.Config) (*cosRemote, error) {
	if cfg == nil {
		return nil, errors.New("COS storage requires config")
	}
	if cfg.COSSecretID == "" || cfg.COSSecretKey == "" {
		return nil, errors.New("COS storage requires GANG_COS_SECRET_ID and GANG_COS_SECRET_KEY")
	}
	bucketURL := strings.TrimSpace(cfg.COSBucketURL)
	if bucketURL == "" {
		if cfg.COSBucket == "" || cfg.COSRegion == "" {
			return nil, errors.New("COS storage requires GANG_COS_BUCKET and GANG_COS_REGION, or GANG_COS_BUCKET_URL")
		}
		bucketURL = fmt.Sprintf("https://%s.cos.%s.myqcloud.com", cfg.COSBucket, cfg.COSRegion)
	}
	u, err := url.Parse(bucketURL)
	if err != nil {
		return nil, fmt.Errorf("parse COS bucket URL: %w", err)
	}
	client := cos.NewClient(
		&cos.BaseURL{BucketURL: u},
		&http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:     cfg.COSSecretID,
				SecretKey:    cfg.COSSecretKey,
				SessionToken: cfg.COSSessionToken,
			},
		},
	)
	return &cosRemote{client: client}, nil
}

func (r *cosRemote) PutFile(ctx context.Context, key, filePath, mimeType, cacheControl string) error {
	options := &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			ContentType:  mimeType,
			CacheControl: cacheControl,
		},
	}
	_, err := r.client.Object.PutFromFile(ctx, key, filePath, options)
	return err
}

func (r *cosRemote) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	resp, err := r.client.Object.Get(ctx, key, nil)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (r *cosRemote) Delete(ctx context.Context, key string) error {
	_, err := r.client.Object.Delete(ctx, key)
	return err
}

func cleanObjectKey(value string) string {
	cleaned := path.Clean(strings.ReplaceAll(strings.TrimSpace(value), "\\", "/"))
	cleaned = strings.Trim(cleaned, "/")
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func escapeObjectKey(key string) string {
	parts := strings.Split(cleanObjectKey(key), "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

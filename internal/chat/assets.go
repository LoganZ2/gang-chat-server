package chat

import (
	"database/sql"
	"errors"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/zhuangkaiyi/gang-chat/server/internal/config"
	"github.com/zhuangkaiyi/gang-chat/server/internal/storage"
)

func RegisterAssetRoutes(r gin.IRouter, db *sql.DB, cfg *config.Config, assetStores ...*storage.AssetStorage) {
	assetStore := firstAssetStore(assetStores)
	if assetStore == nil {
		var err error
		assetStore, err = storage.NewAssetStorage(cfg)
		if err != nil {
			panic(err)
		}
	}
	handler := func(c *gin.Context) {
		assetID := c.Param("asset_id")
		filename := c.Param("filename")
		storageKey, mimeType, err := assetRouteMetadata(db, assetStore, assetID, filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "read asset metadata failed"})
			return
		}
		cachePath, err := assetStore.EnsureCached(c.Request.Context(), storageKey, assetID, filename)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "asset not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "read asset failed"})
			return
		}
		c.Header("Cache-Control", assetStore.CacheControl())
		if mimeType != "" {
			c.Header("Content-Type", mimeType)
		}
		c.File(cachePath)
	}
	r.GET("/assets/:asset_id/:filename", handler)
	r.HEAD("/assets/:asset_id/:filename", handler)
}

func assetRouteMetadata(db *sql.DB, assetStore *storage.AssetStorage, assetID, filename string) (string, string, error) {
	var storageKey sql.NullString
	var mimeType sql.NullString
	err := db.QueryRow(`SELECT storage_key, mime_type FROM assets WHERE id = ? AND filename = ?`, assetID, filename).Scan(&storageKey, &mimeType)
	if err != nil && err != sql.ErrNoRows {
		return "", "", err
	}
	key := ""
	if storageKey.Valid {
		key = storageKey.String
	}
	if key == "" {
		key = assetStore.ObjectKey(assetID, filename)
	}
	mime := ""
	if mimeType.Valid {
		mime = mimeType.String
	}
	return key, mime, nil
}

func (h *Handler) assetStore() *storage.AssetStorage {
	if h != nil && h.Assets != nil {
		return h.Assets
	}
	store, err := storage.NewAssetStorage(h.Cfg)
	if err != nil {
		panic(err)
	}
	return store
}

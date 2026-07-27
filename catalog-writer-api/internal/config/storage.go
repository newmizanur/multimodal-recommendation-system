package config

import (
	"github.com/spf13/viper"

	"github.com/multimodal-recsys/catalog-writer-api/internal/storage"
)

// NewObjectStorage talks to any S3-compatible store. Set S3_ENDPOINT for
// MinIO/R2/etc; leave it unset to use AWS S3 directly.
func NewObjectStorage(v *viper.Viper) (*storage.S3Signer, error) {
	return storage.NewS3Signer(v.GetString("S3_BUCKET"), v.GetString("S3_ENDPOINT"))
}

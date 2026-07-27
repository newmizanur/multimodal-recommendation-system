package usecase

import (
	"context"
	"strconv"

	"github.com/multimodal-recsys/catalog-writer-api/internal/apperror"
	"github.com/multimodal-recsys/catalog-writer-api/internal/dto"
	"github.com/multimodal-recsys/catalog-writer-api/internal/dto/converter"
	"github.com/multimodal-recsys/catalog-writer-api/internal/messaging"
	"github.com/multimodal-recsys/catalog-writer-api/internal/repository"
	"github.com/multimodal-recsys/catalog-writer-api/internal/storage"
)

type productUsecase struct {
	products repository.ProductRepository
	events   *messaging.KafkaProducer
	uploads  *storage.S3Signer
}

func NewProductUsecase(products repository.ProductRepository, events *messaging.KafkaProducer, uploads *storage.S3Signer) ProductUsecase {
	return &productUsecase{products: products, events: events, uploads: uploads}
}

// Create writes the product row, then publishes product.created.
// TODO: wrap the repository write + Kafka publish in a transactional
// outbox if you need stronger delivery guarantees than at-least-once —
// today a crash between the two drops the event silently.
func (u *productUsecase) Create(ctx context.Context, tenantID string, req dto.CreateProductRequest) (dto.ProductResponse, error) {
	product := converter.NewProductFromCreateRequest(tenantID, req)

	created, err := u.products.Create(ctx, product)
	if err != nil {
		return dto.ProductResponse{}, apperror.Internal(err)
	}

	if err := u.events.Publish(ctx, messaging.Event{
		Type:      "product.created",
		TenantID:  tenantID,
		ProductID: strconv.FormatInt(created.ID, 10),
	}); err != nil {
		return dto.ProductResponse{}, apperror.Internal(err)
	}

	return converter.ToProductResponse(created), nil
}

func (u *productUsecase) Update(ctx context.Context, tenantID string, id int64, req dto.UpdateProductRequest) (dto.ProductResponse, error) {
	existing, err := u.products.Get(ctx, tenantID, id)
	if err != nil {
		return dto.ProductResponse{}, apperror.NotFound("product not found")
	}

	converter.ApplyUpdate(&existing, req)

	if err := u.products.Update(ctx, existing); err != nil {
		return dto.ProductResponse{}, apperror.Internal(err)
	}

	if err := u.events.Publish(ctx, messaging.Event{
		Type:      "product.updated",
		TenantID:  tenantID,
		ProductID: strconv.FormatInt(id, 10),
	}); err != nil {
		return dto.ProductResponse{}, apperror.Internal(err)
	}

	return converter.ToProductResponse(existing), nil
}

func (u *productUsecase) Delete(ctx context.Context, tenantID string, id int64) error {
	if err := u.products.Delete(ctx, tenantID, id); err != nil {
		return apperror.Internal(err)
	}

	// The worker hard-deletes the corresponding vector rows on receipt —
	// see worker/internal/usecase/ingest_usecase.go.
	return u.events.Publish(ctx, messaging.Event{
		Type:      "product.deleted",
		TenantID:  tenantID,
		ProductID: strconv.FormatInt(id, 10),
	})
}

// PresignImageUpload issues a presigned S3-compatible PUT URL. The
// client uploads directly to object storage; this API never sees the
// image bytes.
func (u *productUsecase) PresignImageUpload(ctx context.Context, tenantID string, productID int64) (dto.PresignResponse, error) {
	key := tenantID + "/products/" + strconv.FormatInt(productID, 10) + "/upload.jpg" // TODO: content-type aware key/ext
	url, err := u.uploads.PresignPut(ctx, key)
	if err != nil {
		return dto.PresignResponse{}, apperror.Internal(err)
	}
	return dto.PresignResponse{UploadURL: url}, nil
}

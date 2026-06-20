package upload

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Handler struct {
	presignClient *s3.PresignClient
	bucket        string
	publicURL     string
}

func NewHandler(endpoint, keyID, secretKey, bucket, publicURL string) (*Handler, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(keyID, secretKey, "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, opts ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: endpoint}, nil
			},
		)),
	)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) { o.UsePathStyle = true })
	return &Handler{
		presignClient: s3.NewPresignClient(client),
		bucket:        bucket,
		publicURL:     publicURL,
	}, nil
}

type presignResponse struct {
	UploadURL string `json:"upload_url"`
	PublicURL string `json:"public_url"`
	Key       string `json:"key"`
}

func (h *Handler) Presign(w http.ResponseWriter, r *http.Request) {
	ext := r.URL.Query().Get("ext")
	if ext == "" {
		ext = "jpg"
	}
	key := fmt.Sprintf("uploads/%d.%s", time.Now().UnixNano(), ext)
	req, err := h.presignClient.PresignPutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      aws.String(h.bucket),
		Key:         aws.String(key),
		ContentType: aws.String("image/" + ext),
	}, s3.WithPresignExpires(15*time.Minute))
	if err != nil {
		http.Error(w, "could not generate upload URL", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(presignResponse{
		UploadURL: req.URL,
		PublicURL: fmt.Sprintf("%s/%s/%s", h.publicURL, h.bucket, key),
		Key:       key,
	})
}

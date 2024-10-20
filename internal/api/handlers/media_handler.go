package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
)

// FileUploadHandler handles file upload requests
type FileUploadHandler struct {
	S3Client *s3.S3
	Bucket   string
}

// NewFileUploadHandler creates a new FileUploadHandler
func NewFileUploadHandler(region, bucket string) (*FileUploadHandler, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return nil, err
	}

	return &FileUploadHandler{
		S3Client: s3.New(sess),
		Bucket:   bucket,
	}, nil
}

// uploadResponse represents the structure of the upload response
type uploadResponse struct {
	URLs  []string `json:"urls"`
	Error string   `json:"error,omitempty"`
}

// Upload godoc
// @Summary Upload multiple files
// @Description Uploads multiple files to S3 and returns their URLs
// @Tags files
// @Accept multipart/form-data
// @Produce json
// @Param landmarkId path string true "Landmark ID"
// @Param images formData file true "Files to upload"
// @Success 200 {object} uploadResponse
// @Failure 400 {string} string "Invalid request"
// @Failure 500 {string} string "Internal server error"
// @Router /upload/{landmarkId} [post]
func (h *FileUploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	landmarkID := r.URL.Path[len("/upload/"):]
	if landmarkID == "" {
		http.Error(w, "Landmark ID is required", http.StatusBadRequest)
		return
	}

	// Parse the multipart form
	err := r.ParseMultipartForm(32 << 20) // 32 MB max
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		http.Error(w, "No files uploaded", http.StatusBadRequest)
		return
	}

	var urls []string
	for _, fileHeader := range files {
		url, err := h.uploadFile(landmarkID, fileHeader)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		urls = append(urls, url)
	}

	// Return the URLs to the client
	resp := uploadResponse{URLs: urls}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *FileUploadHandler) uploadFile(landmarkID string, fileHeader *multipart.FileHeader) (string, error) {
	// Open the file
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Read the file content
	buffer, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	// Generate a unique filename
	filename := generateUniqueFilename(fileHeader.Filename)

	// Create the S3 key (path)
	key := fmt.Sprintf("landmarks/%s/%s", landmarkID, filename)

	// Upload to S3
	_, err = h.S3Client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(h.Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(buffer),
		ContentType: aws.String(fileHeader.Header.Get("Content-Type")),
	})
	if err != nil {
		return "", err
	}

	// Construct and return the S3 URL
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", h.Bucket, key), nil
}

// SubmitPhotos godoc
// @Summary Submit photos
// @Description Uploads multiple photos to S3 and returns their URLs
// @Tags photos
// @Accept multipart/form-data
// @Produce json
// @Param photos formData file true "Photos to upload"
// @Success 200 {object} uploadResponse
// @Failure 400 {string} string "Invalid request"
// @Failure 500 {string} string "Internal server error"
// @Router /submit-photos [post]
func (h *FileUploadHandler) SubmitPhotos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the multipart form
	err := r.ParseMultipartForm(32 << 20) // 32 MB max
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		http.Error(w, "No photos uploaded", http.StatusBadRequest)
		return
	}

	var urls []string
	for _, fileHeader := range files {
		url, err := h.uploadPhoto(fileHeader)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		urls = append(urls, url)
	}

	// Return the URLs to the client
	resp := uploadResponse{URLs: urls}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *FileUploadHandler) uploadPhoto(fileHeader *multipart.FileHeader) (string, error) {
	// Open the file
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Read the file content
	buffer, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	// Generate a unique filename
	filename := generateUniqueFilename(fileHeader.Filename)

	// Create the S3 key (path)
	key := fmt.Sprintf("user-photos/%s", filename)

	// Upload to S3
	_, err = h.S3Client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(h.Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(buffer),
		ContentType: aws.String(fileHeader.Header.Get("Content-Type")),
	})
	if err != nil {
		return "", err
	}

	// Construct and return the S3 URL
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", h.Bucket, key), nil
}

func generateUniqueFilename(originalFilename string) string {
	extension := filepath.Ext(originalFilename)
	filename := strings.TrimSuffix(originalFilename, extension)
	timestamp := time.Now().Format("20060102150405")
	uniqueID := uuid.New().String()[:8]
	return fmt.Sprintf("%s_%s_%s%s", filename, timestamp, uniqueID, extension)
}

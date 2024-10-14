package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// FileUploadHandler handles file upload requests
type FileUploadHandler struct{}

// NewFileUploadHandler creates a new FileUploadHandler
func NewFileUploadHandler() *FileUploadHandler {
	return &FileUploadHandler{}
}

// uploadResponse represents the structure of the upload response
type uploadResponse struct {
	URL   string `json:"url"`
	Error string `json:"error,omitempty"`
}

// Upload godoc
// @Summary Upload a file
// @Description Uploads a file to S3 and returns its URL
// @Tags files
// @Accept multipart/form-data
// @Produce json
// @Param image formData file true "File to upload"
// @Success 200 {object} uploadResponse
// @Failure 400 {string} string "Invalid request body"
// @Failure 500 {string} string "Internal server error"
// @Router /upload [post]
func (h *FileUploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get the file from the form
	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Define a maximum allowed size (e.g., 10 MB)
	const MaxAllowedSize = 10 << 20 // 10 MB

	// Validate the file size
	if header.Size > MaxAllowedSize {
		http.Error(w, "File size exceeds the maximum allowed limit", http.StatusBadRequest)
		return
	}

	// Read the file content
	buffer := make([]byte, header.Size)
	_, err = file.Read(buffer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Initialize AWS session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("eu-north-1"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create S3 service client
	svc := s3.New(sess)

	// Upload to S3
	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String("properties-photos"),
		Key:         aws.String(header.Filename),
		Body:        bytes.NewReader(buffer),
		ContentType: aws.String(header.Header.Get("Content-Type")),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Construct the S3 URL
	s3URL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
		"properties-photos",
		"eu-north-1",
		header.Filename,
	)

	// Return the URL to the client
	resp := uploadResponse{URL: s3URL}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

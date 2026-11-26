package aionui

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	dtoaionui "github.com/QuantumNous/new-api/dto/aionui"
	"github.com/QuantumNous/new-api/model"
	agentplatform "github.com/QuantumNous/new-api/service/agentplatform"
)

const clientLogUploadArtifactKind = "aionui-client-log"

type ClientLogUploadLimits struct {
	MaxFileBytes  int64
	MaxTotalBytes int64
	MaxFiles      int
	UploadTimeout time.Duration
}

type ClientLogUploadInput struct {
	ClientVersion string
	Date          string
	Email         string
	Reader        *multipart.Reader
	UserID        int
}

type ClientLogUploadError struct {
	Code       string
	HTTPStatus int
}

func (e ClientLogUploadError) Error() string {
	return e.Code
}

type clientLogUploadFile struct {
	bytes  int64
	name   string
	path   string
	sha256 string
}

type ClientLogUploadService struct {
	limits      ClientLogUploadLimits
	newStore    func() (agentplatform.ArtifactStore, error)
	newTempFile func() (*os.File, error)
}

func DefaultClientLogUploadLimits() ClientLogUploadLimits {
	return ClientLogUploadLimits{
		MaxFileBytes:  common.AionUiClientLogMaxFileBytes,
		MaxTotalBytes: common.AionUiClientLogMaxTotalBytes,
		MaxFiles:      common.AionUiClientLogMaxFiles,
		UploadTimeout: time.Duration(common.AionUiClientLogUploadTimeoutSeconds) * time.Second,
	}
}

func NewClientLogUploadService(limits ClientLogUploadLimits) *ClientLogUploadService {
	return &ClientLogUploadService{
		limits:      limits,
		newStore:    agentplatform.DefaultArtifactStore,
		newTempFile: func() (*os.File, error) { return os.CreateTemp("", "new-api-aionui-client-log-*") },
	}
}

func (s *ClientLogUploadService) Upload(ctx context.Context, input ClientLogUploadInput) (dtoaionui.ClientLogUploadResponse, error) {
	if s == nil || input.Reader == nil || input.UserID <= 0 || !validClientLogUploadLimits(s.limits) {
		return dtoaionui.ClientLogUploadResponse{}, ClientLogUploadError{Code: "storage_unavailable", HTTPStatus: http.StatusServiceUnavailable}
	}
	date, err := time.Parse("2006-01-02", input.Date)
	if err != nil || date.Format("2006-01-02") != input.Date {
		return dtoaionui.ClientLogUploadResponse{}, clientLogValidationError("invalid_date")
	}
	if !validClientLogVersion(input.ClientVersion) || !validClientLogEmail(input.Email) {
		return dtoaionui.ClientLogUploadResponse{}, clientLogValidationError("invalid_request")
	}
	uploadContext, cancel := context.WithTimeout(ctx, s.limits.UploadTimeout)
	defer cancel()

	files := make([]clientLogUploadFile, 0)
	defer func() {
		for _, file := range files {
			if file.path != "" {
				_ = os.Remove(file.path)
			}
		}
	}()
	seenNames := make(map[string]struct{})
	var totalBytes int64
	for {
		if uploadContext.Err() != nil {
			return dtoaionui.ClientLogUploadResponse{}, ClientLogUploadError{Code: "storage_unavailable", HTTPStatus: http.StatusBadGateway}
		}
		part, partErr := input.Reader.NextPart()
		if errors.Is(partErr, io.EOF) {
			break
		}
		if partErr != nil {
			if isClientLogPayloadTooLarge(partErr) {
				return dtoaionui.ClientLogUploadResponse{}, clientLogPayloadTooLargeError()
			}
			return dtoaionui.ClientLogUploadResponse{}, clientLogValidationError("invalid_multipart")
		}
		if part.FormName() != "files" || strings.HasPrefix(strings.ToLower(part.Header.Get("Content-Type")), "multipart/") {
			part.Close()
			return dtoaionui.ClientLogUploadResponse{}, clientLogValidationError("invalid_multipart")
		}
		filename, filenameErr := rawMultipartFilename(part)
		if filenameErr != nil || !validClientLogFilename(filename) {
			part.Close()
			return dtoaionui.ClientLogUploadResponse{}, clientLogValidationError("invalid_filename")
		}
		if _, exists := seenNames[filename]; exists {
			part.Close()
			return dtoaionui.ClientLogUploadResponse{}, clientLogValidationError("duplicate_filename")
		}
		if len(files) >= s.limits.MaxFiles {
			part.Close()
			return dtoaionui.ClientLogUploadResponse{}, clientLogPayloadTooLargeError()
		}

		file, fileErr := s.newTempFile()
		if fileErr != nil {
			part.Close()
			return dtoaionui.ClientLogUploadResponse{}, ClientLogUploadError{Code: "storage_unavailable", HTTPStatus: http.StatusServiceUnavailable}
		}
		filePath := file.Name()
		hash := sha256.New()
		written, copyErr := io.Copy(io.MultiWriter(file, hash), io.LimitReader(part, s.limits.MaxFileBytes+1))
		closeErr := file.Close()
		part.Close()
		if copyErr != nil || closeErr != nil {
			_ = os.Remove(filePath)
			if isClientLogPayloadTooLarge(copyErr) {
				return dtoaionui.ClientLogUploadResponse{}, clientLogPayloadTooLargeError()
			}
			return dtoaionui.ClientLogUploadResponse{}, clientLogValidationError("invalid_multipart")
		}
		if written == 0 {
			_ = os.Remove(filePath)
			return dtoaionui.ClientLogUploadResponse{}, clientLogValidationError("empty_file")
		}
		if written > s.limits.MaxFileBytes || totalBytes > s.limits.MaxTotalBytes-written {
			_ = os.Remove(filePath)
			return dtoaionui.ClientLogUploadResponse{}, clientLogPayloadTooLargeError()
		}
		totalBytes += written
		seenNames[filename] = struct{}{}
		files = append(files, clientLogUploadFile{bytes: written, name: filename, path: filePath, sha256: hex.EncodeToString(hash.Sum(nil))})
	}
	if len(files) == 0 {
		return dtoaionui.ClientLogUploadResponse{}, clientLogValidationError("empty_batch")
	}

	store, err := s.newStore()
	if err != nil {
		return dtoaionui.ClientLogUploadResponse{}, ClientLogUploadError{Code: "storage_unavailable", HTTPStatus: http.StatusBadGateway}
	}
	results := make([]dtoaionui.ClientLogUploadFile, 0, len(files))
	for index := range files {
		file := &files[index]
		key := strings.Join([]string{
			"client-logs",
			model.NormalizeEmail(input.Email),
			input.ClientVersion,
			date.Format("2006"),
			date.Format("01"),
			date.Format("02"),
			file.name,
		}, "/")
		_, putErr := store.PutFile(uploadContext, agentplatform.PutArtifactInput{
			Kind:        clientLogUploadArtifactKind,
			BucketKey:   key,
			LocalPath:   file.path,
			ContentType: "application/octet-stream",
			Sha256:      file.sha256,
			SizeBytes:   file.bytes,
			Metadata: map[string]string{
				"artifact-kind":  clientLogUploadArtifactKind,
				"user-id":        fmt.Sprintf("%d", input.UserID),
				"client-version": input.ClientVersion,
				"log-date":       input.Date,
				"sha256":         file.sha256,
			},
		})
		if putErr != nil {
			return dtoaionui.ClientLogUploadResponse{}, ClientLogUploadError{Code: "storage_unavailable", HTTPStatus: http.StatusBadGateway}
		}
		if err := os.Remove(file.path); err != nil {
			common.SysError("AionUI client log upload temporary file cleanup failed")
		} else {
			file.path = ""
		}
		results = append(results, dtoaionui.ClientLogUploadFile{Name: file.name, Bytes: file.bytes, Sha256: file.sha256})
	}
	return dtoaionui.ClientLogUploadResponse{
		Date:          input.Date,
		ClientVersion: input.ClientVersion,
		FileCount:     len(results),
		TotalBytes:    totalBytes,
		Files:         results,
	}, nil
}

func validClientLogUploadLimits(limits ClientLogUploadLimits) bool {
	return limits.MaxFileBytes > 0 && limits.MaxTotalBytes >= limits.MaxFileBytes && limits.MaxFiles > 0 && limits.UploadTimeout > 0
}

func validClientLogVersion(value string) bool {
	if len(value) == 0 || len(value) > 64 {
		return false
	}
	for index, character := range value {
		if (index == 0 && !(character >= '0' && character <= '9' || character >= 'A' && character <= 'Z' || character >= 'a' && character <= 'z')) ||
			(index > 0 && !(character >= '0' && character <= '9' || character >= 'A' && character <= 'Z' || character >= 'a' && character <= 'z' || strings.ContainsRune("._+-", character))) {
			return false
		}
	}
	return true
}

func validClientLogEmail(value string) bool {
	value = model.NormalizeEmail(value)
	if value == "" || len(value) > 320 || strings.ContainsAny(value, "/\\\x00") {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	parsed, err := mail.ParseAddress(value)
	return err == nil && parsed.Address == value
}

func validClientLogFilename(value string) bool {
	if value == "" || len(value) > 255 || !utf8.ValidString(value) || value == "." || value == ".." || strings.ContainsAny(value, "/\\\x00") {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func rawMultipartFilename(part *multipart.Part) (string, error) {
	_, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
	if err != nil {
		return "", err
	}
	filename, ok := params["filename"]
	if !ok {
		return "", errors.New("multipart filename missing")
	}
	return filename, nil
}

func clientLogValidationError(code string) ClientLogUploadError {
	return ClientLogUploadError{Code: code, HTTPStatus: http.StatusBadRequest}
}

func clientLogPayloadTooLargeError() ClientLogUploadError {
	return ClientLogUploadError{Code: "payload_too_large", HTTPStatus: http.StatusRequestEntityTooLarge}
}

func isClientLogPayloadTooLarge(err error) bool {
	var maxBytesError *http.MaxBytesError
	return errors.As(err, &maxBytesError)
}

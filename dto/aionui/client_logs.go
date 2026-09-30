package aionui

type ClientLogUploadFile struct {
	Name   string `json:"name"`
	Bytes  int64  `json:"bytes"`
	Sha256 string `json:"sha256"`
}

type ClientLogUploadResponse struct {
	Date          string                `json:"date"`
	ClientVersion string                `json:"client_version"`
	FileCount     int                   `json:"file_count"`
	TotalBytes    int64                 `json:"total_bytes"`
	Files         []ClientLogUploadFile `json:"files"`
}

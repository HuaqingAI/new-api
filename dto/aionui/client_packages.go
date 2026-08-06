package aionui

type ClientPackageQuery struct {
	Platform *string `form:"platform" json:"platform,omitempty"`
	Status   *string `form:"status" json:"status,omitempty"`
	Page     *int    `form:"page" json:"page,omitempty"`
	PageSize *int    `form:"page_size" json:"page_size,omitempty"`
}

type ClientPackageStatusRequest struct {
	Status string `json:"status"`
}

type ClientPackageItem struct {
	Id                     int     `json:"id"`
	Platform               string  `json:"platform"`
	Version                string  `json:"version"`
	Status                 string  `json:"status"`
	FileName               string  `json:"file_name"`
	FileSha256             string  `json:"file_sha256"`
	FileSha512             string  `json:"file_sha512"`
	FileSize               int64   `json:"file_size"`
	UpdateFileName         string  `json:"update_file_name,omitempty"`
	UpdateFileSha256       string  `json:"update_file_sha256,omitempty"`
	UpdateFileSha512       string  `json:"update_file_sha512,omitempty"`
	UpdateFileSize         int64   `json:"update_file_size,omitempty"`
	UpdateMetadataFileName string  `json:"update_metadata_file_name,omitempty"`
	UpdateMetadataSha256   string  `json:"update_metadata_sha256,omitempty"`
	ReleaseNote            string  `json:"release_note,omitempty"`
	CreatedBy              int     `json:"created_by"`
	PublishedAt            *string `json:"published_at,omitempty"`
	CreatedAt              string  `json:"created_at"`
	UpdatedAt              string  `json:"updated_at"`
}

type ClientPackageListResponse struct {
	Items    []ClientPackageItem `json:"items"`
	Total    int                 `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}

type ClientPackageLatestItem struct {
	Platform  string             `json:"platform"`
	Available bool               `json:"available"`
	Release   *ClientPackageItem `json:"release"`
}

type ClientPackageLatestResponse struct {
	Items []ClientPackageLatestItem `json:"items"`
}

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

type ClientPackageScopeInput struct {
	SubjectType string `json:"subject_type"`
	SubjectId   string `json:"subject_id"`
}

type ClientPackageRolloutRequest struct {
	RolloutMode string                    `json:"rollout_mode"`
	Scopes      []ClientPackageScopeInput `json:"scopes"`
}

type ClientUpdateAccessRequest struct {
	Platform        string `json:"platform"`
	CurrentVersion  string `json:"current_version"`
	ExpectedVersion string `json:"expected_version"`
}

type ClientUpdateAccessRelease struct {
	Version  string `json:"version"`
	Platform string `json:"platform"`
}

type ClientUpdateAccessResponse struct {
	LegacyOpen         bool                       `json:"legacy_open,omitempty"`
	Eligible           bool                       `json:"eligible"`
	Release            *ClientUpdateAccessRelease `json:"release,omitempty"`
	ArtifactCapability string                     `json:"artifact_capability,omitempty"`
	ExpiresAt          int64                      `json:"expires_at,omitempty"`
}

type ClientPackageDirectUploadFileRequest struct {
	Kind     string `json:"kind"`
	FileName string `json:"file_name"`
	Sha256   string `json:"sha256"`
	Sha512   string `json:"sha512"`
	Size     int64  `json:"size"`
}

type ClientPackageDirectUploadInitRequest struct {
	Platform string                                 `json:"platform"`
	Version  string                                 `json:"version"`
	Publish  bool                                   `json:"publish"`
	Files    []ClientPackageDirectUploadFileRequest `json:"files"`
}

type ClientPackageDirectUploadTarget struct {
	Kind        string            `json:"kind"`
	FileName    string            `json:"file_name"`
	ObjectURI   string            `json:"object_uri"`
	ObjectKey   string            `json:"object_key"`
	UploadURL   string            `json:"upload_url"`
	ContentType string            `json:"content_type"`
	Headers     map[string]string `json:"headers"`
	ExpiresAt   int64             `json:"expires_at"`
	Sha256      string            `json:"sha256"`
	Sha512      string            `json:"sha512"`
	Size        int64             `json:"size"`
}

type ClientPackageDirectUploadInitResponse struct {
	Files []ClientPackageDirectUploadTarget `json:"files"`
}

type ClientPackageDirectUploadCompleteRequest struct {
	Platform           string                          `json:"platform"`
	Version            string                          `json:"version"`
	ReleaseNote        string                          `json:"release_note"`
	Publish            bool                            `json:"publish"`
	RolloutMode        string                          `json:"rollout_mode"`
	Scopes             []ClientPackageScopeInput       `json:"scopes"`
	File               ClientPackageDirectUploadTarget `json:"file"`
	UpdateFile         ClientPackageDirectUploadTarget `json:"update_file"`
	UpdateMetadataFile ClientPackageDirectUploadTarget `json:"update_metadata_file"`
}

type ClientPackageItem struct {
	Id                     int     `json:"id"`
	Platform               string  `json:"platform"`
	Version                string  `json:"version"`
	Status                 string  `json:"status"`
	RolloutMode            string  `json:"rollout_mode"`
	ScopeCount             int     `json:"scope_count"`
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

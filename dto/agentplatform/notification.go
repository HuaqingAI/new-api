package agentplatform

type NotificationWriteRequest struct {
	Title   *string `json:"title"`
	Summary *string `json:"summary"`
	Content *string `json:"content"`
	Enabled *bool   `json:"enabled"`
}

type NotificationItem struct {
	Id          string `json:"id"`
	Title       string `json:"title"`
	Summary     string `json:"summary"`
	Enabled     bool   `json:"enabled"`
	Version     int64  `json:"version"`
	PublishedAt int64  `json:"published_at"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
	CreatedBy   int    `json:"created_by"`
	UpdatedBy   int    `json:"updated_by"`
}

type NotificationDetail struct {
	NotificationItem
	Content string `json:"content"`
}

type NotificationList struct {
	Items    []NotificationItem `json:"items"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

type ClientNotificationItem struct {
	Id          string `json:"id"`
	Title       string `json:"title"`
	Summary     string `json:"summary"`
	Version     int64  `json:"version"`
	PublishedAt int64  `json:"published_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

type ClientNotificationDetail struct {
	ClientNotificationItem
	Content string `json:"content"`
}

type ClientNotificationList struct {
	ViewerScope string                   `json:"viewer_scope"`
	Items       []ClientNotificationItem `json:"items"`
	Total       int                      `json:"total"`
}

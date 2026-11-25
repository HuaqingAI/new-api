package agentplatform

// Notification is a global client announcement. Reading state belongs to AionUi.
type Notification struct {
	Id             int64  `gorm:"primaryKey;index:idx_ap_notification_updated,priority:2"`
	NotificationId string `gorm:"type:varchar(40);not null;uniqueIndex:uq_ap_notification_id;index:idx_ap_notification_feed,priority:3"`
	Title          string `gorm:"type:varchar(120);not null"`
	Summary        string `gorm:"type:varchar(240);not null"`
	Content        string `gorm:"type:text;not null"`
	Enabled        bool   `gorm:"not null;index:idx_ap_notification_feed,priority:1"`
	Version        int64  `gorm:"not null"`
	PublishedAt    int64  `gorm:"not null;index:idx_ap_notification_feed,priority:2"`
	CreatedAt      int64  `gorm:"not null;autoCreateTime:false"`
	UpdatedAt      int64  `gorm:"not null;autoUpdateTime:false;index:idx_ap_notification_updated,priority:1"`
	CreatedBy      int    `gorm:"not null"`
	UpdatedBy      int    `gorm:"not null"`
}

func (Notification) TableName() string {
	return "agent_platform_notifications"
}

package aionui

import "time"

type ClientInstallation struct {
	Id                  int       `json:"id" gorm:"primaryKey"`
	DeviceId            string    `json:"-" gorm:"type:varchar(128);not null;uniqueIndex:uq_aionui_client_installations_device"`
	UserId              int       `json:"user_id" gorm:"type:int;not null;index:idx_aionui_client_installations_user"`
	Username            string    `json:"username" gorm:"type:varchar(64);not null"`
	Email               string    `json:"email" gorm:"type:varchar(320);not null;index:idx_aionui_client_installations_email"`
	DisplayName         string    `json:"display_name" gorm:"type:varchar(128);not null"`
	DepartmentNamesJSON string    `json:"-" gorm:"type:text;not null"`
	ClientVersion       string    `json:"client_version" gorm:"type:varchar(64);not null;index:idx_aionui_client_installations_version;index:idx_aionui_client_installations_platform_version,priority:2"`
	Platform            string    `json:"platform" gorm:"type:varchar(32);not null;index:idx_aionui_client_installations_platform_version,priority:1"`
	LanIP               string    `json:"lan_ip" gorm:"type:varchar(45);not null"`
	FirstHeartbeatAt    time.Time `json:"first_heartbeat_at" gorm:"not null"`
	LastHeartbeatAt     time.Time `json:"last_heartbeat_at" gorm:"not null;index:idx_aionui_client_installations_last_heartbeat"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (ClientInstallation) TableName() string {
	return "agent_client_installations"
}

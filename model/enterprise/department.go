package enterprise

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"gorm.io/gorm"
)

type DepartmentNameHistoryEntry struct {
	Name      string `json:"name"`
	ChangedAt int64  `json:"changed_at"`
}

type Department struct {
	Id          int    `json:"id" gorm:"primaryKey"`
	TenantId    int    `json:"tenant_id" gorm:"not null;default:0;index:idx_departments_tenant"`
	Name        string `json:"name" gorm:"type:varchar(255);not null"`
	ParentId    *int   `json:"parent_id" gorm:"index:idx_departments_parent;check:chk_departments_parent_not_self,parent_id IS NULL OR parent_id <> id"`
	Status      int    `json:"status" gorm:"not null;default:1;index:idx_departments_status"`
	SourceType  int    `json:"source_type" gorm:"not null;default:1;index:idx_departments_source_external"`
	ExternalId  string `json:"external_id" gorm:"type:varchar(128);not null;default:'';index:idx_departments_source_external"`
	SyncStatus  int    `json:"sync_status" gorm:"not null;default:0"`
	SyncError   string `json:"sync_error" gorm:"type:varchar(1024);not null;default:''"`
	NameHistory string `json:"name_history" gorm:"type:text;not null"`
	CreatedAt   int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt   int64  `json:"updated_at" gorm:"bigint"`
	DeletedAt   int64  `json:"deleted_at" gorm:"bigint;not null;default:0"`
}

func (Department) TableName() string {
	return "enterprise_departments"
}

func (d *Department) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().Unix()
	if d.CreatedAt == 0 {
		d.CreatedAt = now
	}
	if d.UpdatedAt == 0 {
		d.UpdatedAt = now
	}
	if d.Status == 0 {
		d.Status = constant.DepartmentStatusEnabled
	}
	if d.SourceType == 0 {
		d.SourceType = constant.DepartmentSourceTypeManual
	}
	if d.NameHistory == "" {
		if err := d.SetNameHistory(nil); err != nil {
			return err
		}
	}
	return nil
}

func (d *Department) BeforeUpdate(tx *gorm.DB) error {
	d.UpdatedAt = time.Now().Unix()
	return nil
}

func (d *Department) SetNameHistory(entries []DepartmentNameHistoryEntry) error {
	if entries == nil {
		entries = []DepartmentNameHistoryEntry{}
	}
	data, err := common.Marshal(entries)
	if err != nil {
		return err
	}
	d.NameHistory = string(data)
	return nil
}

func (d Department) ParsedNameHistory() ([]DepartmentNameHistoryEntry, error) {
	if d.NameHistory == "" {
		return []DepartmentNameHistoryEntry{}, nil
	}
	var entries []DepartmentNameHistoryEntry
	if err := common.UnmarshalJsonStr(d.NameHistory, &entries); err != nil {
		return nil, err
	}
	if entries == nil {
		return []DepartmentNameHistoryEntry{}, nil
	}
	return entries, nil
}

package agentplatform

import (
	"errors"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	dtoagentplatform "github.com/QuantumNous/new-api/dto/agentplatform"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"gorm.io/gorm"
)

var (
	ErrInvalidNotification  = errors.New("invalid notification")
	ErrNotificationNotFound = errors.New("notification not found")
	ErrNotificationConflict = errors.New("notification version conflict")
)

type NotificationInput struct {
	Title   string
	Summary string
	Content string
	Enabled bool
}

type NotificationService struct {
	db *gorm.DB
}

func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{db: db}
}

func (s *NotificationService) List(page, pageSize int, status string) (dtoagentplatform.NotificationList, error) {
	result := dtoagentplatform.NotificationList{Items: []dtoagentplatform.NotificationItem{}, Page: page, PageSize: pageSize}
	if page < 1 || pageSize < 1 || pageSize > 100 || page-1 > math.MaxInt/pageSize {
		return result, ErrInvalidNotification
	}
	query := s.db.Model(&apmodel.Notification{})
	switch status {
	case "all":
	case "enabled":
		query = query.Where("enabled = ?", true)
	case "disabled":
		query = query.Where("enabled = ?", false)
	default:
		return result, ErrInvalidNotification
	}
	if err := query.Count(&result.Total).Error; err != nil {
		return result, err
	}
	var records []apmodel.Notification
	if err := query.Omit("content").Order("updated_at DESC").Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records).Error; err != nil {
		return result, err
	}
	for _, record := range records {
		result.Items = append(result.Items, NotificationManagementItem(record))
	}
	return result, nil
}

func (s *NotificationService) Get(id string, enabledOnly bool) (apmodel.Notification, error) {
	var record apmodel.Notification
	query := s.db.Where("notification_id = ?", id)
	if enabledOnly {
		query = query.Where("enabled = ?", true)
	}
	if err := query.First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return record, ErrNotificationNotFound
		}
		return record, err
	}
	if enabledOnly && record.PublishedAt <= 0 {
		return apmodel.Notification{}, errors.New("enabled notification has no publication time")
	}
	return record, nil
}

func (s *NotificationService) ListEnabled() ([]dtoagentplatform.ClientNotificationItem, error) {
	items := []dtoagentplatform.ClientNotificationItem{}
	var records []apmodel.Notification
	if err := s.db.Select("notification_id", "title", "summary", "version", "published_at", "updated_at").Where("enabled = ?", true).
		Order("published_at DESC").Order("notification_id DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	for _, record := range records {
		if record.PublishedAt <= 0 {
			return nil, errors.New("enabled notification has no publication time")
		}
		items = append(items, NotificationClientItem(record))
	}
	return items, nil
}

func (s *NotificationService) Create(input NotificationInput, actorID int) (apmodel.Notification, error) {
	if err := input.Normalize(); err != nil || actorID <= 0 {
		return apmodel.Notification{}, ErrInvalidNotification
	}
	now := time.Now().Unix()
	record := apmodel.Notification{
		NotificationId: "ntf_" + common.GetUUID(), Title: input.Title, Summary: input.Summary, Content: input.Content,
		Enabled: input.Enabled, Version: 1, CreatedAt: now, UpdatedAt: now, CreatedBy: actorID, UpdatedBy: actorID,
	}
	if record.Enabled {
		record.PublishedAt = now
	}
	err := s.db.Create(&record).Error
	return record, err
}

func (s *NotificationService) Update(id string, version int64, input NotificationInput, actorID int) (apmodel.Notification, error) {
	if err := input.Normalize(); err != nil {
		return apmodel.Notification{}, err
	}
	return s.change(id, version, &input, input.Enabled, actorID)
}

func (s *NotificationService) SetEnabled(id string, version int64, enabled bool, actorID int) (apmodel.Notification, error) {
	return s.change(id, version, nil, enabled, actorID)
}

func (s *NotificationService) change(id string, version int64, input *NotificationInput, enabled bool, actorID int) (apmodel.Notification, error) {
	if actorID <= 0 || version < 1 || version == math.MaxInt64 {
		return apmodel.Notification{}, ErrInvalidNotification
	}
	record, err := s.Get(id, false)
	if err != nil {
		return record, err
	}
	if record.Version != version {
		return record, ErrNotificationConflict
	}
	if record.Enabled == enabled && (input == nil || (record.Title == input.Title && record.Summary == input.Summary && record.Content == input.Content)) {
		return record, nil
	}
	if input != nil {
		record.Title, record.Summary, record.Content = input.Title, input.Summary, input.Content
	}
	record.Enabled = enabled
	record.Version++
	record.UpdatedAt, record.UpdatedBy = time.Now().Unix(), actorID
	if record.Enabled && record.PublishedAt == 0 {
		record.PublishedAt = record.UpdatedAt
	}
	result := s.db.Model(&apmodel.Notification{}).Where("notification_id = ? AND version = ?", id, version).Updates(map[string]any{
		"title": record.Title, "summary": record.Summary, "content": record.Content, "enabled": record.Enabled,
		"version": record.Version, "published_at": record.PublishedAt, "updated_at": record.UpdatedAt, "updated_by": record.UpdatedBy,
	})
	if result.Error != nil {
		return apmodel.Notification{}, result.Error
	}
	if result.RowsAffected == 0 {
		if _, err := s.Get(id, false); err != nil {
			return apmodel.Notification{}, err
		}
		return apmodel.Notification{}, ErrNotificationConflict
	}
	return record, nil
}

func (s *NotificationService) Delete(id string, version int64) error {
	result := s.db.Where("notification_id = ? AND version = ?", id, version).Delete(&apmodel.Notification{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		if _, err := s.Get(id, false); err != nil {
			return err
		}
		return ErrNotificationConflict
	}
	return nil
}

func (input *NotificationInput) Normalize() error {
	if !utf8.ValidString(input.Title) || !utf8.ValidString(input.Summary) || !utf8.ValidString(input.Content) {
		return ErrInvalidNotification
	}
	input.Title, input.Summary = strings.TrimSpace(input.Title), strings.TrimSpace(input.Summary)
	input.Content = strings.ReplaceAll(input.Content, "\r\n", "\n")
	if input.Title == "" || utf8.RuneCountInString(input.Title) > 120 || utf8.RuneCountInString(input.Summary) > 240 ||
		strings.TrimSpace(input.Content) == "" || utf8.RuneCountInString(input.Content) > 10000 || len(input.Content) > 40*1024 {
		return ErrInvalidNotification
	}
	for _, r := range input.Title {
		if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
			return ErrInvalidNotification
		}
	}
	for _, text := range []string{input.Summary, input.Content} {
		for _, r := range text {
			if unicode.IsControl(r) && r != '\n' && r != '\t' {
				return ErrInvalidNotification
			}
		}
	}
	return nil
}

func NotificationManagementItem(record apmodel.Notification) dtoagentplatform.NotificationItem {
	return dtoagentplatform.NotificationItem{
		Id: record.NotificationId, Title: record.Title, Summary: record.Summary, Enabled: record.Enabled,
		Version: record.Version, PublishedAt: record.PublishedAt, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt,
		CreatedBy: record.CreatedBy, UpdatedBy: record.UpdatedBy,
	}
}

func NotificationClientItem(record apmodel.Notification) dtoagentplatform.ClientNotificationItem {
	return dtoagentplatform.ClientNotificationItem{
		Id: record.NotificationId, Title: record.Title, Summary: record.Summary,
		Version: record.Version, PublishedAt: record.PublishedAt, UpdatedAt: record.UpdatedAt,
	}
}

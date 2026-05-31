package enterprise

import (
	"strings"

	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

func lookupUserIDsByUsername(db *gorm.DB, username string) ([]int, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return []int{}, nil
	}
	if db == nil {
		db = model.DB
	}
	var userIDs []int
	if err := db.Session(&gorm.Session{NewDB: true}).Model(&model.User{}).Where("username = ?", username).Pluck("id", &userIDs).Error; err != nil {
		return nil, err
	}
	if userIDs == nil {
		userIDs = []int{}
	}
	return userIDs, nil
}

func loadCurrentUsernames(db *gorm.DB, userIDs []int) (map[int]string, error) {
	if len(userIDs) == 0 {
		return map[int]string{}, nil
	}
	if db == nil {
		db = model.DB
	}
	var rows []struct {
		Id       int
		Username string
	}
	if err := db.Session(&gorm.Session{NewDB: true}).Model(&model.User{}).Select("id, username").Where("id IN ?", userIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[int]string, len(rows))
	for _, row := range rows {
		out[row.Id] = row.Username
	}
	return out, nil
}

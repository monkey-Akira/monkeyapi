package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	RiskAlertStatusOpen    = "open"
	RiskAlertStatusHandled = "handled"
	RiskAlertStatusIgnored = "ignored"

	userRiskIpThreshold = 3
)

type UserLoginIpLog struct {
	Id        int    `json:"id"`
	UserId    int    `json:"user_id" gorm:"index"`
	Ip        string `json:"ip" gorm:"type:varchar(64);index"`
	CreatedAt int64  `json:"created_at" gorm:"index"`
}

type UserRiskAlert struct {
	Id            int    `json:"id"`
	Ip            string `json:"ip" gorm:"type:varchar(64);uniqueIndex"`
	UserCount     int    `json:"user_count"`
	RegisterCount int    `json:"register_count"`
	LoginCount    int    `json:"login_count"`
	Reason        string `json:"reason" gorm:"type:text"`
	Status        string `json:"status" gorm:"type:varchar(20);index;default:'open'"`
	CreatedAt     int64  `json:"created_at" gorm:"index"`
	UpdatedAt     int64  `json:"updated_at" gorm:"index"`
}

type RiskAlertUser struct {
	Id              int    `json:"id"`
	Username        string `json:"username"`
	DisplayName     string `json:"display_name"`
	Email           string `json:"email"`
	Status          int    `json:"status"`
	Role            int    `json:"role"`
	Quota           int    `json:"quota"`
	UsedQuota       int    `json:"used_quota"`
	RequestCount    int    `json:"request_count"`
	RegisterIp      string `json:"register_ip"`
	LastLoginIp     string `json:"last_login_ip"`
	CreatedAt       int64  `json:"created_at"`
	LastLoginAt     int64  `json:"last_login_at"`
	RegisterMatched bool   `json:"register_matched"`
	LoginMatched    bool   `json:"login_matched"`
}

func normalizeUserRiskIp(ip string) string {
	return strings.TrimSpace(ip)
}

func RecordUserLoginIp(userId int, ip string) error {
	ip = normalizeUserRiskIp(ip)
	if userId <= 0 || ip == "" {
		return nil
	}
	now := common.GetTimestamp()
	if err := DB.Create(&UserLoginIpLog{
		UserId:    userId,
		Ip:        ip,
		CreatedAt: now,
	}).Error; err != nil {
		return err
	}
	return CheckAndUpsertIpRiskAlert(ip)
}

func CheckAndUpsertIpRiskAlert(ip string) error {
	if !IsUserIpRiskAlertEnabled() {
		return nil
	}
	ip = normalizeUserRiskIp(ip)
	if ip == "" {
		return nil
	}

	registerUserIds, loginUserIds, err := getRiskUserIdsByIp(ip)
	if err != nil {
		return err
	}
	userCount := countDistinctInts(registerUserIds, loginUserIds)
	if userCount < userRiskIpThreshold {
		return nil
	}

	now := common.GetTimestamp()
	reason := fmt.Sprintf("IP associated with %d users (registration: %d, login: %d)", userCount, len(registerUserIds), len(loginUserIds))
	var alert UserRiskAlert
	err = DB.Where("ip = ?", ip).First(&alert).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return DB.Create(&UserRiskAlert{
			Ip:            ip,
			UserCount:     userCount,
			RegisterCount: len(registerUserIds),
			LoginCount:    len(loginUserIds),
			Reason:        reason,
			Status:        RiskAlertStatusOpen,
			CreatedAt:     now,
			UpdatedAt:     now,
		}).Error
	}
	if err != nil {
		return err
	}

	status := alert.Status
	if status == "" || (status == RiskAlertStatusHandled && userCount > alert.UserCount) {
		status = RiskAlertStatusOpen
	}
	return DB.Model(&UserRiskAlert{}).Where("id = ?", alert.Id).Updates(map[string]interface{}{
		"user_count":     userCount,
		"register_count": len(registerUserIds),
		"login_count":    len(loginUserIds),
		"reason":         reason,
		"status":         status,
		"updated_at":     now,
	}).Error
}

func GetUserRiskAlerts(status string, pageInfo *common.PageInfo) ([]UserRiskAlert, int64, error) {
	status = strings.TrimSpace(status)
	query := DB.Model(&UserRiskAlert{})
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var alerts []UserRiskAlert
	if err := query.Order("updated_at desc").Offset(pageInfo.GetStartIdx()).Limit(pageInfo.GetPageSize()).Find(&alerts).Error; err != nil {
		return nil, 0, err
	}
	return alerts, total, nil
}

func GetUserRiskAlertDetail(id int) (*UserRiskAlert, []RiskAlertUser, error) {
	var alert UserRiskAlert
	if err := DB.Where("id = ?", id).First(&alert).Error; err != nil {
		return nil, nil, err
	}
	registerUserIds, loginUserIds, err := getRiskUserIdsByIp(alert.Ip)
	if err != nil {
		return nil, nil, err
	}
	userIdSet := make(map[int]struct{})
	registerSet := make(map[int]struct{})
	loginSet := make(map[int]struct{})
	for _, id := range registerUserIds {
		userIdSet[id] = struct{}{}
		registerSet[id] = struct{}{}
	}
	for _, id := range loginUserIds {
		userIdSet[id] = struct{}{}
		loginSet[id] = struct{}{}
	}
	userIds := make([]int, 0, len(userIdSet))
	for id := range userIdSet {
		userIds = append(userIds, id)
	}
	sort.Ints(userIds)
	if len(userIds) == 0 {
		return &alert, []RiskAlertUser{}, nil
	}

	var users []User
	if err := DB.Where("id IN ?", userIds).Order("id asc").Find(&users).Error; err != nil {
		return nil, nil, err
	}
	items := make([]RiskAlertUser, 0, len(users))
	for _, user := range users {
		_, registerMatched := registerSet[user.Id]
		_, loginMatched := loginSet[user.Id]
		items = append(items, RiskAlertUser{
			Id:              user.Id,
			Username:        user.Username,
			DisplayName:     user.DisplayName,
			Email:           user.Email,
			Status:          user.Status,
			Role:            user.Role,
			Quota:           user.Quota,
			UsedQuota:       user.UsedQuota,
			RequestCount:    user.RequestCount,
			RegisterIp:      user.RegisterIp,
			LastLoginIp:     user.LastLoginIp,
			CreatedAt:       user.CreatedAt,
			LastLoginAt:     user.LastLoginAt,
			RegisterMatched: registerMatched,
			LoginMatched:    loginMatched,
		})
	}
	return &alert, items, nil
}

func UpdateUserRiskAlertStatus(id int, status string) error {
	status = strings.TrimSpace(status)
	switch status {
	case RiskAlertStatusOpen, RiskAlertStatusHandled, RiskAlertStatusIgnored:
	default:
		return fmt.Errorf("invalid risk alert status")
	}
	return DB.Model(&UserRiskAlert{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": common.GetTimestamp(),
	}).Error
}

func IsUserIpRiskAlertEnabled() bool {
	common.OptionMapRWMutex.RLock()
	value, ok := common.OptionMap["UserIpRiskAlertEnabled"]
	common.OptionMapRWMutex.RUnlock()
	if !ok {
		return true
	}
	return value == "true"
}

func getRiskUserIdsByIp(ip string) ([]int, []int, error) {
	var registerIds []int
	if err := DB.Model(&User{}).Where("register_ip = ?", ip).Pluck("id", &registerIds).Error; err != nil {
		return nil, nil, err
	}
	registerIds = uniqueSortedInts(registerIds)

	loginSet := make(map[int]struct{})
	var logLoginIds []int
	if err := DB.Model(&UserLoginIpLog{}).Where("ip = ?", ip).Pluck("user_id", &logLoginIds).Error; err != nil {
		return nil, nil, err
	}
	for _, id := range logLoginIds {
		if id > 0 {
			loginSet[id] = struct{}{}
		}
	}
	var lastLoginIds []int
	if err := DB.Model(&User{}).Where("last_login_ip = ?", ip).Pluck("id", &lastLoginIds).Error; err != nil {
		return nil, nil, err
	}
	for _, id := range lastLoginIds {
		if id > 0 {
			loginSet[id] = struct{}{}
		}
	}
	loginIds := make([]int, 0, len(loginSet))
	for id := range loginSet {
		loginIds = append(loginIds, id)
	}
	sort.Ints(loginIds)
	return registerIds, loginIds, nil
}

func countDistinctInts(groups ...[]int) int {
	seen := make(map[int]struct{})
	for _, group := range groups {
		for _, id := range group {
			if id > 0 {
				seen[id] = struct{}{}
			}
		}
	}
	return len(seen)
}

func uniqueSortedInts(values []int) []int {
	seen := make(map[int]struct{})
	for _, value := range values {
		if value > 0 {
			seen[value] = struct{}{}
		}
	}
	result := make([]int, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Ints(result)
	return result
}

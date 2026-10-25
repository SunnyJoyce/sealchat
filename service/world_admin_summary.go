package service

import (
	"sync"
	"time"

	"sealchat/model"
)

// AdminWorldSummary 是平台状态页中展示的世界基本信息，只包含世界层面的概要与计数，不含频道/消息内容。
type AdminWorldSummary struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Visibility      string    `json:"visibility"`
	OwnerID         string    `json:"ownerId"`
	OwnerNickname   string    `json:"ownerNickname"`
	CreatedAt       time.Time `json:"createdAt"`
	MemberCount     int64     `json:"memberCount"`
	ChannelCount    int64     `json:"channelCount"`
	LastActiveAt    int64     `json:"lastActiveAt"` // 毫秒；0 表示尚无消息
	MessageChars    int64     `json:"messageChars"`
	MessageCharsIC  int64     `json:"messageCharsIc"`
	MessageCharsOOC int64     `json:"messageCharsOoc"`
}

const adminWorldSummaryCacheTTL = time.Minute

var adminWorldSummaryCache struct {
	mu      sync.Mutex
	items   []AdminWorldSummary
	expires time.Time
}

// AdminListWorldSummaries 返回全部正常状态世界（含私有世界）的概要信息，结果缓存一分钟。
func AdminListWorldSummaries() ([]AdminWorldSummary, error) {
	adminWorldSummaryCache.mu.Lock()
	defer adminWorldSummaryCache.mu.Unlock()
	if adminWorldSummaryCache.items != nil && time.Now().Before(adminWorldSummaryCache.expires) {
		return adminWorldSummaryCache.items, nil
	}
	items, err := loadAdminWorldSummaries()
	if err != nil {
		return nil, err
	}
	adminWorldSummaryCache.items = items
	adminWorldSummaryCache.expires = time.Now().Add(adminWorldSummaryCacheTTL)
	return items, nil
}

func loadAdminWorldSummaries() ([]AdminWorldSummary, error) {
	db := model.GetDB()
	inactive := []string{"deleted", "archived"}

	var worlds []model.WorldModel
	if err := db.Model(&model.WorldModel{}).
		Select("id, name, description, visibility, owner_id, created_at").
		Where("status NOT IN ?", inactive).
		Order("created_at ASC, id ASC").
		Find(&worlds).Error; err != nil {
		return nil, err
	}
	items := make([]AdminWorldSummary, 0, len(worlds))
	if len(worlds) == 0 {
		return items, nil
	}

	worldIDs := make([]string, 0, len(worlds))
	ownerIDs := make([]string, 0, len(worlds))
	for _, w := range worlds {
		worldIDs = append(worldIDs, w.ID)
		if w.OwnerID != "" {
			ownerIDs = append(ownerIDs, w.OwnerID)
		}
	}

	ownerNames := map[string]string{}
	if len(ownerIDs) > 0 {
		var owners []model.UserModel
		if err := db.Model(&model.UserModel{}).Select("id, nickname, username").Where("id IN ?", ownerIDs).Find(&owners).Error; err != nil {
			return nil, err
		}
		for _, u := range owners {
			name := u.Nickname
			if name == "" {
				name = u.Username
			}
			ownerNames[u.ID] = name
		}
	}

	type memberRow struct {
		WorldID string
		Total   int64
	}
	var memberRows []memberRow
	if err := db.Model(&model.WorldMemberModel{}).
		Select("world_id, COUNT(*) AS total").
		Where("world_id IN ?", worldIDs).
		Group("world_id").
		Scan(&memberRows).Error; err != nil {
		return nil, err
	}
	memberCounts := make(map[string]int64, len(memberRows))
	for _, row := range memberRows {
		memberCounts[row.WorldID] = row.Total
	}

	type channelRow struct {
		WorldID      string
		Total        int64
		LastActiveAt int64
	}
	var channelRows []channelRow
	if err := db.Model(&model.ChannelModel{}).
		Select("world_id, COUNT(*) AS total, COALESCE(MAX(recent_sent_at), 0) AS last_active_at").
		Where("world_id IN ? AND is_private = ? AND status NOT IN ?", worldIDs, false, []string{model.ChannelStatusDeleted, model.ChannelStatusArchived}).
		Group("world_id").
		Scan(&channelRows).Error; err != nil {
		return nil, err
	}
	channelStats := make(map[string]channelRow, len(channelRows))
	for _, row := range channelRows {
		channelStats[row.WorldID] = row
	}

	charRows, err := model.CountWorldMessageChars()
	if err != nil {
		return nil, err
	}
	charStats := make(map[string]model.WorldMessageCharStats, len(charRows))
	for _, row := range charRows {
		charStats[row.WorldID] = row
	}

	for _, w := range worlds {
		chars := charStats[w.ID]
		channels := channelStats[w.ID]
		items = append(items, AdminWorldSummary{
			ID:              w.ID,
			Name:            w.Name,
			Description:     w.Description,
			Visibility:      w.Visibility,
			OwnerID:         w.OwnerID,
			OwnerNickname:   ownerNames[w.OwnerID],
			CreatedAt:       w.CreatedAt,
			MemberCount:     memberCounts[w.ID],
			ChannelCount:    channels.Total,
			LastActiveAt:    channels.LastActiveAt,
			MessageChars:    chars.Chars,
			MessageCharsIC:  chars.CharsIC,
			MessageCharsOOC: chars.CharsOOC,
		})
	}
	return items, nil
}

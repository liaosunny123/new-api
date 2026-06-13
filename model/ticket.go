package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// Ticket priorities
const (
	TicketPriorityLow    = "low"
	TicketPriorityMedium = "medium"
	TicketPriorityHigh   = "high"
)

// Ticket statuses
const (
	TicketStatusOpen       = "open"
	TicketStatusProcessing = "processing"
	TicketStatusClosed     = "closed"
)

// MaxTicketsPerDay limits how many tickets a single user can create within a rolling 24h window.
const MaxTicketsPerDay = 10

// Ticket represents a support ticket submitted by a user.
type Ticket struct {
	Id          int            `json:"id"`
	UserId      int            `json:"user_id" gorm:"index"`
	Username    string         `json:"username" gorm:"-:all"` // only for api response (admin list)
	Title       string         `json:"title" gorm:"type:varchar(255)"`
	Content     string         `json:"content" gorm:"type:text"`
	Priority    string         `json:"priority" gorm:"type:varchar(20);default:'medium'"`
	Status      string         `json:"status" gorm:"type:varchar(20);default:'open';index"`
	CreatedTime int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`

	Messages []TicketMessage `json:"messages,omitempty" gorm:"-:all"` // populated on detail fetch
}

// TicketMessage is a single message in a ticket conversation thread.
type TicketMessage struct {
	Id          int    `json:"id"`
	TicketId    int    `json:"ticket_id" gorm:"index"`
	UserId      int    `json:"user_id"`
	IsAdmin     bool   `json:"is_admin"`
	Content     string `json:"content" gorm:"type:text"`
	CreatedTime int64  `json:"created_time" gorm:"bigint"`
}

func ValidTicketPriority(p string) bool {
	switch p {
	case TicketPriorityLow, TicketPriorityMedium, TicketPriorityHigh:
		return true
	}
	return false
}

func ValidTicketStatus(s string) bool {
	switch s {
	case TicketStatusOpen, TicketStatusProcessing, TicketStatusClosed:
		return true
	}
	return false
}

// CountUserTicketsSince counts how many tickets a user created since the given unix timestamp.
func CountUserTicketsSince(userId int, sinceTs int64) (int64, error) {
	var count int64
	err := DB.Model(&Ticket{}).
		Where("user_id = ? AND created_time >= ?", userId, sinceTs).
		Count(&count).Error
	return count, err
}

// CreateTicket creates a ticket together with its first message (the user's opening content).
func CreateTicket(userId int, title, content, priority string) (*Ticket, error) {
	if !ValidTicketPriority(priority) {
		priority = TicketPriorityMedium
	}
	now := common.GetTimestamp()
	ticket := &Ticket{
		UserId:      userId,
		Title:       title,
		Content:     content,
		Priority:    priority,
		Status:      TicketStatusOpen,
		CreatedTime: now,
		UpdatedTime: now,
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(ticket).Error; err != nil {
			return err
		}
		msg := &TicketMessage{
			TicketId:    ticket.Id,
			UserId:      userId,
			IsAdmin:     false,
			Content:     content,
			CreatedTime: now,
		}
		return tx.Create(msg).Error
	})
	if err != nil {
		return nil, err
	}
	return ticket, nil
}

// GetUserTickets returns a paginated list of a user's own tickets (newest first).
func GetUserTickets(userId int, startIdx, num int) ([]*Ticket, int64, error) {
	var tickets []*Ticket
	var total int64
	if err := DB.Model(&Ticket{}).Where("user_id = ?", userId).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := DB.Where("user_id = ?", userId).
		Order("updated_time desc").Limit(num).Offset(startIdx).Find(&tickets).Error
	return tickets, total, err
}

// GetAllTickets returns a paginated list of all tickets (admin), optionally filtered by status/priority.
func GetAllTickets(status, priority string, startIdx, num int) ([]*Ticket, int64, error) {
	var tickets []*Ticket
	var total int64
	query := DB.Model(&Ticket{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if priority != "" {
		query = query.Where("priority = ?", priority)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("updated_time desc").Limit(num).Offset(startIdx).Find(&tickets).Error
	if err != nil {
		return nil, 0, err
	}
	fillTicketUsernames(tickets)
	return tickets, total, nil
}

// fillTicketUsernames populates the (non-persisted) Username field for admin listing.
func fillTicketUsernames(tickets []*Ticket) {
	if len(tickets) == 0 {
		return
	}
	idSet := make(map[int]struct{})
	ids := make([]int, 0, len(tickets))
	for _, t := range tickets {
		if _, ok := idSet[t.UserId]; !ok {
			idSet[t.UserId] = struct{}{}
			ids = append(ids, t.UserId)
		}
	}
	type idName struct {
		Id       int
		Username string
	}
	var rows []idName
	if err := DB.Model(&User{}).Select("id, username").Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return
	}
	nameMap := make(map[int]string, len(rows))
	for _, r := range rows {
		nameMap[r.Id] = r.Username
	}
	for _, t := range tickets {
		t.Username = nameMap[t.UserId]
	}
}

// GetTicketById fetches a single ticket. When withMessages is true, its thread is loaded.
func GetTicketById(id int, withMessages bool) (*Ticket, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	ticket := &Ticket{}
	if err := DB.First(ticket, "id = ?", id).Error; err != nil {
		return nil, err
	}
	if withMessages {
		messages, err := GetTicketMessages(id)
		if err != nil {
			return nil, err
		}
		ticket.Messages = messages
	}
	return ticket, nil
}

// GetTicketMessages returns all messages of a ticket ordered chronologically.
func GetTicketMessages(ticketId int) ([]TicketMessage, error) {
	var messages []TicketMessage
	err := DB.Where("ticket_id = ?", ticketId).Order("id asc").Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// AddTicketMessage appends a message to a ticket thread and bumps its updated time.
// When the replier is an admin and newStatus is a valid status, the ticket status is updated;
// when a user replies to a closed ticket, it is reopened.
func AddTicketMessage(ticketId, userId int, isAdmin bool, content, newStatus string) (*TicketMessage, error) {
	now := common.GetTimestamp()
	msg := &TicketMessage{
		TicketId:    ticketId,
		UserId:      userId,
		IsAdmin:     isAdmin,
		Content:     content,
		CreatedTime: now,
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		var ticket Ticket
		if err := tx.First(&ticket, "id = ?", ticketId).Error; err != nil {
			return err
		}
		if err := tx.Create(msg).Error; err != nil {
			return err
		}
		updates := map[string]interface{}{"updated_time": now}
		if isAdmin {
			if ValidTicketStatus(newStatus) {
				updates["status"] = newStatus
			} else if ticket.Status == TicketStatusOpen {
				updates["status"] = TicketStatusProcessing
			}
		} else if ticket.Status == TicketStatusClosed {
			// user replied to a closed ticket -> reopen
			updates["status"] = TicketStatusOpen
		}
		return tx.Model(&Ticket{}).Where("id = ?", ticketId).Updates(updates).Error
	})
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// UpdateTicketStatus sets a ticket's status.
func UpdateTicketStatus(ticketId int, status string) error {
	if !ValidTicketStatus(status) {
		return errors.New("invalid ticket status")
	}
	return DB.Model(&Ticket{}).Where("id = ?", ticketId).
		Updates(map[string]interface{}{
			"status":       status,
			"updated_time": common.GetTimestamp(),
		}).Error
}

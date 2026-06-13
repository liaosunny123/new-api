package controller

import (
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

type createTicketRequest struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	Priority string `json:"priority"`
}

type replyTicketRequest struct {
	Content string `json:"content"`
	Status  string `json:"status"` // admin only, optional
}

type updateTicketStatusRequest struct {
	Status string `json:"status"`
}

// ListUserTickets returns the current user's own tickets (paginated).
func ListUserTickets(c *gin.Context) {
	userId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)
	tickets, total, err := model.GetUserTickets(userId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(tickets)
	common.ApiSuccess(c, pageInfo)
}

// GetUserTicket returns one of the current user's tickets with its full thread.
func GetUserTicket(c *gin.Context) {
	userId := c.GetInt("id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	ticket, err := model.GetTicketById(id, true)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgTicketNotFound)
		return
	}
	if ticket.UserId != userId {
		common.ApiErrorI18n(c, i18n.MsgTicketNoPermission)
		return
	}
	common.ApiSuccess(c, ticket)
}

// CreateTicket lets a user open a new ticket, enforcing the daily cap.
func CreateTicket(c *gin.Context) {
	userId := c.GetInt("id")
	var req createTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)
	if req.Title == "" {
		common.ApiErrorI18n(c, i18n.MsgTicketTitleEmpty)
		return
	}
	if req.Content == "" {
		common.ApiErrorI18n(c, i18n.MsgTicketContentEmpty)
		return
	}
	if req.Priority == "" {
		req.Priority = model.TicketPriorityMedium
	}
	if !model.ValidTicketPriority(req.Priority) {
		common.ApiErrorI18n(c, i18n.MsgTicketInvalidPriority)
		return
	}
	if len(req.Title) > 255 {
		req.Title = req.Title[:255]
	}

	// Enforce daily limit (rolling 24h window).
	since := time.Now().Add(-24 * time.Hour).Unix()
	count, err := model.CountUserTicketsSince(userId, since)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if count >= model.MaxTicketsPerDay {
		common.ApiErrorI18n(c, i18n.MsgTicketDailyLimit, map[string]any{"Max": model.MaxTicketsPerDay})
		return
	}

	ticket, err := model.CreateTicket(userId, req.Title, req.Content, req.Priority)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, ticket)
}

// ReplyUserTicket lets a user add a message to their own ticket.
func ReplyUserTicket(c *gin.Context) {
	userId := c.GetInt("id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	var req replyTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		common.ApiErrorI18n(c, i18n.MsgTicketReplyEmpty)
		return
	}
	ticket, err := model.GetTicketById(id, false)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgTicketNotFound)
		return
	}
	if ticket.UserId != userId {
		common.ApiErrorI18n(c, i18n.MsgTicketNoPermission)
		return
	}
	msg, err := model.AddTicketMessage(id, userId, false, req.Content, "")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, msg)
}

// AdminListTickets returns all tickets (paginated, filterable by status/priority).
func AdminListTickets(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	status := c.Query("status")
	priority := c.Query("priority")
	tickets, total, err := model.GetAllTickets(status, priority, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(tickets)
	common.ApiSuccess(c, pageInfo)
}

// AdminGetTicket returns any ticket with its full thread.
func AdminGetTicket(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	ticket, err := model.GetTicketById(id, true)
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgTicketNotFound)
		return
	}
	common.ApiSuccess(c, ticket)
}

// AdminReplyTicket lets an admin reply to a ticket and optionally set its status.
func AdminReplyTicket(c *gin.Context) {
	userId := c.GetInt("id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	var req replyTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		common.ApiErrorI18n(c, i18n.MsgTicketReplyEmpty)
		return
	}
	if req.Status != "" && !model.ValidTicketStatus(req.Status) {
		common.ApiErrorI18n(c, i18n.MsgTicketInvalidStatus)
		return
	}
	if _, err := model.GetTicketById(id, false); err != nil {
		common.ApiErrorI18n(c, i18n.MsgTicketNotFound)
		return
	}
	msg, err := model.AddTicketMessage(id, userId, true, req.Content, req.Status)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, msg)
}

// AdminUpdateTicketStatus updates a ticket's status without adding a message.
func AdminUpdateTicketStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	var req updateTicketStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if !model.ValidTicketStatus(req.Status) {
		common.ApiErrorI18n(c, i18n.MsgTicketInvalidStatus)
		return
	}
	if _, err := model.GetTicketById(id, false); err != nil {
		common.ApiErrorI18n(c, i18n.MsgTicketNotFound)
		return
	}
	if err := model.UpdateTicketStatus(id, req.Status); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccessI18n(c, i18n.MsgUpdateSuccess, nil)
}

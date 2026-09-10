package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/services"
	"backend/middleware"
	"backend/render"
)

// ============================================================
// CHAT HANDLER
//
// Responsibility:
// - Handles Agro-Shield general chat.
// - Displays the user's conversations.
// - Opens private/group conversations.
// - Sends messages.
// - Creates groups.
// - Adds members.
// - Removes members.
// - Allows members to leave groups.
// - Deletes messages.
//
// IMPORTANT:
//
// This handler is DIFFERENT from NegotiationHandler.
//
// NegotiationHandler:
//     Buyer ↔ Seller
//     Price offers
//     Accept / Reject
//
// ChatHandler:
//     User ↔ User
//     User ↔ Group
//     Normal communication
// ============================================================

// ChatHandler handles general Agro-Shield chat functionality.
type ChatHandler struct {
	service *services.ChatService
	users   *repository.UserRepository
}

// ============================================================
// CREATE CHAT HANDLER
// ============================================================

// NewChatHandler creates a new ChatHandler using the provided
// chat service and user repository.
func NewChatHandler(
	service *services.ChatService,
	users *repository.UserRepository,
) *ChatHandler {
	return &ChatHandler{
		service: service,
		users:   users,
	}
}

// ============================================================
// CHAT LIST PAGE DATA
//
// Used by:
//
// GET /chat
//
// Shows all conversations belonging to the logged-in user.
// ============================================================

// ChatListPageData contains the data rendered on the main chat page.
type ChatListPageData struct {
	User          interface{}
	Conversations []models.Conversation
	UnreadCount   int
	Users         []models.User
	Query         string
	Error         string
}

// chatPresenceResponse contains recently active users for the
// client-side chat presence indicator.
type chatPresenceResponse struct {
	OnlineUserIDs []int `json:"online_user_ids"`
}

// ============================================================
// CHAT PRESENCE
// ============================================================

// PresenceHandler returns recently active chat users for client-side
// presence updates.
func (h *ChatHandler) PresenceHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	user, ok := middleware.UserFromContext(r)

	if !ok || user == nil {
		http.Error(
			w,
			"Unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	middleware.MarkUserOnline(user.ID)

	users, err := h.users.ListForChat(
		user.ID,
		"",
	)

	if err != nil {
		http.Error(
			w,
			"Could not load presence",
			http.StatusInternalServerError,
		)
		return
	}

	onlineIDs := make([]int, 0)

	for _, chatUser := range users {
		if middleware.IsUserOnline(chatUser.ID) {
			onlineIDs = append(
				onlineIDs,
				chatUser.ID,
			)
		}
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.Header().Set(
		"Cache-Control",
		"no-store",
	)

	if err := json.NewEncoder(w).Encode(
		chatPresenceResponse{
			OnlineUserIDs: onlineIDs,
		},
	); err != nil {
		log.Println(
			"presence response error:",
			err,
		)
	}
}

// ============================================================
// CHAT THREAD PAGE DATA
//
// Used by:
//
// GET /chat/view?id=123
//
// Contains the conversation and its messages.
// ============================================================

// ChatThreadPageData contains the data rendered on a conversation
// thread page.
type ChatThreadPageData struct {
	User         interface{}
	Conversation *models.Conversation
	Messages     []models.ChatMessage
	Members      []models.ConversationMember
	Error        string
}

// ============================================================
// CHAT HOME
//
// GET /chat
//
// Shows the user's conversations.
// ============================================================

// ListHandler displays the authenticated user's conversations
// and available users for starting a new chat.
func (h *ChatHandler) ListHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	user, ok := middleware.UserFromContext(r)

	if !ok || user == nil {
		http.Redirect(
			w,
			r,
			"/login",
			http.StatusSeeOther,
		)
		return
	}

	conversations, err := h.service.MyConversations(
		user.ID,
	)

	if err != nil {
		log.Println(
			"failed to load conversations:",
			err,
		)

		h.renderList(
			w,
			user,
			nil,
			err.Error(),
		)
		return
	}

	query := strings.TrimSpace(
		r.URL.Query().Get("q"),
	)

	users, usersErr := h.users.ListForChat(
		user.ID,
		query,
	)

	if usersErr != nil {
		h.renderList(
			w,
			user,
			conversations,
			usersErr.Error(),
		)
		return
	}

	data := ChatListPageData{
		User:          user,
		Conversations: conversations,
		Users:         users,
		Query:         query,
	}

	for _, conversation := range conversations {
		data.UnreadCount += conversation.UnreadCount
	}

	if err := h.renderTemplate(
		w,
		"chat.html",
		data,
	); err != nil {
		log.Println(
			"chat render error:",
			err,
		)

		// Do not attempt another HTTP response here because the
		// template renderer may already have written the response.
		return
	}
}

// ============================================================
// VIEW CONVERSATION
//
// GET /chat/view?id=123
//
// Shows:
//
// - Conversation
// - Messages
// - Members
//
// The service checks that the user belongs to the conversation.
// ============================================================

// ViewHandler displays a conversation, its messages, and its members.
func (h *ChatHandler) ViewHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	user, ok := middleware.UserFromContext(r)

	if !ok || user == nil {
		http.Redirect(
			w,
			r,
			"/login",
			http.StatusSeeOther,
		)
		return
	}

	conversationID, err := strconv.Atoi(
		r.URL.Query().Get("id"),
	)

	if err != nil || conversationID <= 0 {
		http.Error(
			w,
			"Invalid conversation ID",
			http.StatusBadRequest,
		)
		return
	}

	conversation, err := h.service.GetConversation(
		conversationID,
		user.ID,
	)

	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusForbidden,
		)
		return
	}

	messages, err := h.service.GetMessages(
		conversationID,
		user.ID,
	)

	if err != nil {
		log.Println(
			"failed to load chat messages:",
			err,
		)

		http.Error(
			w,
			"Failed to load messages",
			http.StatusInternalServerError,
		)
		return
	}

	members, err := h.service.GetMembers(
		conversationID,
		user.ID,
	)

	if err != nil {
		log.Println(
			"failed to load conversation members:",
			err,
		)

		http.Error(
			w,
			"Failed to load members",
			http.StatusInternalServerError,
		)
		return
	}

	data := ChatThreadPageData{
		User:         user,
		Conversation: conversation,
		Messages:     messages,
		Members:      members,
	}

	if err := h.renderTemplate(
		w,
		"chat_thread.html",
		data,
	); err != nil {
		log.Println(
			"chat thread render error:",
			err,
		)

		// Avoid a second response after template rendering has
		// potentially already written to the response.
		return
	}
}

// ============================================================
// SEND MESSAGE
//
// POST /chat/send
//
// Form:
//
// conversation_id
// message
// ============================================================

// SendMessageHandler sends a normal chat message to a conversation.
func (h *ChatHandler) SendMessageHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	user, ok := middleware.UserFromContext(r)

	if !ok || user == nil {
		http.Redirect(
			w,
			r,
			"/login",
			http.StatusSeeOther,
		)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	conversationID, err := strconv.Atoi(
		r.FormValue("conversation_id"),
	)

	if err != nil || conversationID <= 0 {
		http.Error(
			w,
			"Invalid conversation ID",
			http.StatusBadRequest,
		)
		return
	}

	message := strings.TrimSpace(
		r.FormValue("message"),
	)

	if message == "" {
		http.Redirect(
			w,
			r,
			"/chat/view?id="+strconv.Itoa(conversationID),
			http.StatusSeeOther,
		)
		return
	}

	_, err = h.service.SendMessage(
		conversationID,
		user.ID,
		message,
	)

	if err != nil {
		log.Println(
			"failed to send chat message:",
			err,
		)

		http.Redirect(
			w,
			r,
			"/chat/view?id="+strconv.Itoa(conversationID)+
				"&error="+
				err.Error(),
			http.StatusSeeOther,
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/chat/view?id="+strconv.Itoa(conversationID),
		http.StatusSeeOther,
	)
}

// ============================================================
// CREATE GROUP
//
// POST /chat/group/create
//
// Form:
//
// name
// ============================================================

// CreateGroupHandler creates a new group conversation with the
// authenticated user as the creator.
func (h *ChatHandler) CreateGroupHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	user, ok := middleware.UserFromContext(r)

	if !ok || user == nil {
		http.Redirect(
			w,
			r,
			"/login",
			http.StatusSeeOther,
		)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	name := strings.TrimSpace(
		r.FormValue("name"),
	)

	conversation, err := h.service.CreateGroup(
		user.ID,
		name,
	)

	if err != nil {
		log.Println(
			"failed to create group:",
			err,
		)

		h.renderList(
			w,
			user,
			nil,
			err.Error(),
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/chat/view?id="+strconv.Itoa(
			conversation.ID,
		),
		http.StatusSeeOther,
	)
}

// ============================================================
// CREATE PRIVATE CHAT
//
// POST /chat/private
//
// Form:
//
// user_id
// ============================================================

// CreatePrivateChatHandler creates or opens a private conversation
// between the authenticated user and another user.
func (h *ChatHandler) CreatePrivateChatHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	user, ok := middleware.UserFromContext(r)

	if !ok || user == nil {
		http.Redirect(
			w,
			r,
			"/login",
			http.StatusSeeOther,
		)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	otherUserID, err := strconv.Atoi(
		r.FormValue("user_id"),
	)

	if err != nil || otherUserID <= 0 {
		http.Error(
			w,
			"Invalid user ID",
			http.StatusBadRequest,
		)
		return
	}

	conversation, err := h.service.CreatePrivateChat(
		user.ID,
		otherUserID,
	)

	if err != nil {
		log.Println(
			"failed to create private chat:",
			err,
		)

		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/chat/view?id="+strconv.Itoa(
			conversation.ID,
		),
		http.StatusSeeOther,
	)
}

// ============================================================
// ADD MEMBER
//
// POST /chat/member/add
//
// Form:
//
// conversation_id
// user_id
//
// The service is responsible for verifying membership and
// conversation permissions.
// ============================================================

// AddMemberHandler adds a user to a group conversation.
func (h *ChatHandler) AddMemberHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	user, ok := middleware.UserFromContext(r)

	if !ok || user == nil {
		http.Redirect(
			w,
			r,
			"/login",
			http.StatusSeeOther,
		)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	conversationID, err := strconv.Atoi(
		r.FormValue("conversation_id"),
	)

	if err != nil || conversationID <= 0 {
		http.Error(
			w,
			"Invalid conversation ID",
			http.StatusBadRequest,
		)
		return
	}

	userID, err := strconv.Atoi(
		r.FormValue("user_id"),
	)

	if err != nil || userID <= 0 {
		http.Error(
			w,
			"Invalid user ID",
			http.StatusBadRequest,
		)
		return
	}

	err = h.service.AddMember(
		conversationID,
		user.ID,
		userID,
	)

	if err != nil {
		log.Println(
			"failed to add member:",
			err,
		)

		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/chat/view?id="+strconv.Itoa(conversationID),
		http.StatusSeeOther,
	)
}

// ============================================================
// REMOVE MEMBER
//
// POST /chat/member/remove
//
// Only the group creator can remove another member.
// ============================================================

// RemoveMemberHandler removes a user from a group conversation.
func (h *ChatHandler) RemoveMemberHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	user, ok := middleware.UserFromContext(r)

	if !ok || user == nil {
		http.Redirect(
			w,
			r,
			"/login",
			http.StatusSeeOther,
		)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	conversationID, err := strconv.Atoi(
		r.FormValue("conversation_id"),
	)

	if err != nil || conversationID <= 0 {
		http.Error(
			w,
			"Invalid conversation ID",
			http.StatusBadRequest,
		)
		return
	}

	userID, err := strconv.Atoi(
		r.FormValue("user_id"),
	)

	if err != nil || userID <= 0 {
		http.Error(
			w,
			"Invalid user ID",
			http.StatusBadRequest,
		)
		return
	}

	err = h.service.RemoveMember(
		conversationID,
		user.ID,
		userID,
	)

	if err != nil {
		log.Println(
			"failed to remove member:",
			err,
		)

		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/chat/view?id="+strconv.Itoa(conversationID),
		http.StatusSeeOther,
	)
}

// ============================================================
// LEAVE GROUP
//
// POST /chat/leave
//
// Normal members can leave.
//
// The creator cannot leave until ownership transfer is added.
// ============================================================

// LeaveGroupHandler allows the authenticated user to leave a group.
func (h *ChatHandler) LeaveGroupHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	user, ok := middleware.UserFromContext(r)

	if !ok || user == nil {
		http.Redirect(
			w,
			r,
			"/login",
			http.StatusSeeOther,
		)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	conversationID, err := strconv.Atoi(
		r.FormValue("conversation_id"),
	)

	if err != nil || conversationID <= 0 {
		http.Error(
			w,
			"Invalid conversation ID",
			http.StatusBadRequest,
		)
		return
	}

	err = h.service.LeaveGroup(
		conversationID,
		user.ID,
	)

	if err != nil {
		log.Println(
			"failed to leave group:",
			err,
		)

		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/chat",
		http.StatusSeeOther,
	)
}

// ============================================================
// DELETE MESSAGE
//
// POST /chat/message/delete
//
// Only the sender can delete their own message.
// ============================================================

// DeleteMessageHandler deletes a chat message belonging to the
// authenticated user.
func (h *ChatHandler) DeleteMessageHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	user, ok := middleware.UserFromContext(r)

	if !ok || user == nil {
		http.Redirect(
			w,
			r,
			"/login",
			http.StatusSeeOther,
		)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	messageID, err := strconv.Atoi(
		r.FormValue("message_id"),
	)

	if err != nil || messageID <= 0 {
		http.Error(
			w,
			"Invalid message ID",
			http.StatusBadRequest,
		)
		return
	}

	err = h.service.DeleteMessage(
		messageID,
		user.ID,
	)

	if err != nil {
		log.Println(
			"failed to delete message:",
			err,
		)

		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	conversationID := r.FormValue(
		"conversation_id",
	)

	if conversationID == "" {
		http.Redirect(
			w,
			r,
			"/chat",
			http.StatusSeeOther,
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/chat/view?id="+conversationID,
		http.StatusSeeOther,
	)
}

// ============================================================
// RENDER CHAT LIST
// ============================================================

// renderList renders the main chat list page.
func (h *ChatHandler) renderList(
	w http.ResponseWriter,
	user interface{},
	conversations []models.Conversation,
	errorMessage string,
) {
	data := ChatListPageData{
		User:          user,
		Conversations: conversations,
		Error:         errorMessage,
	}

	for _, conversation := range conversations {
		data.UnreadCount += conversation.UnreadCount
	}

	if err := h.renderTemplate(
		w,
		"chat.html",
		data,
	); err != nil {
		log.Println(
			"chat list render error:",
			err,
		)

		// Avoid writing a second HTTP response if the template
		// renderer has already written the response.
		return
	}
}

// ============================================================
// renderTemplate renders a chat page through the shared renderer. This keeps
// template paths independent of the process working directory and prevents a
// partial response when template execution fails.
func (h *ChatHandler) renderTemplate(
	w http.ResponseWriter,
	templateName string,
	data interface{},
) error {
	return render.RenderTemplates(w, templateName, data)
}

package services

import (
	"errors"
	"strings"

	"backend/internal/models"
	"backend/internal/repository"
)

// ChatService controls the business rules for Agro-Shield chat.
//
// The service sits between HTTP handlers and repositories:
//
// Handler
//
//	↓
//
// ChatService
//
//	↓
//
// Repository
//
//	↓
//
// # PostgreSQL
//
// ChatService is responsible for protecting conversations,
// membership, messages, and chat access.
type ChatService struct {
	conversationRepo *repository.ConversationRepository
	memberRepo       *repository.ConversationMemberRepository
	messageRepo      *repository.ChatMessageRepository
	notification     *NotificationService
}

// NewChatService creates a new ChatService connected to the
// conversation, member, message, and notification services.
func NewChatService(
	conversationRepo *repository.ConversationRepository,
	memberRepo *repository.ConversationMemberRepository,
	messageRepo *repository.ChatMessageRepository,
	notification *NotificationService,
) *ChatService {
	return &ChatService{
		conversationRepo: conversationRepo,
		memberRepo:       memberRepo,
		messageRepo:      messageRepo,
		notification:     notification,
	}
}

// CreateGroup creates a new group conversation.
//
// The creator automatically becomes the first member of the group.
func (s *ChatService) CreateGroup(
	creatorID int,
	name string,
) (*models.Conversation, error) {
	if creatorID <= 0 {
		return nil, errors.New("invalid creator ID")
	}

	name = strings.TrimSpace(name)

	if name == "" {
		return nil, errors.New("group name cannot be empty")
	}

	if len(name) > 100 {
		return nil, errors.New("group name is too long")
	}

	conversation := &models.Conversation{
		Title:            &name,
		ConversationType: "group",
		CreatedBy:        &creatorID,
	}

	if err := s.conversationRepo.Create(
		conversation,
	); err != nil {
		return nil, err
	}

	// The creator automatically joins the group.
	member := &models.ConversationMember{
		ConversationID: conversation.ID,
		UserID:         creatorID,
	}

	if err := s.memberRepo.AddMember(member); err != nil {
		return nil, err
	}

	return conversation, nil
}

// CreatePrivateChat creates a one-to-one conversation between
// two different users.
//
// If a private conversation already exists between the two users,
// the existing conversation is returned instead of creating a duplicate.
func (s *ChatService) CreatePrivateChat(
	userID,
	otherUserID int,
) (*models.Conversation, error) {
	if userID <= 0 || otherUserID <= 0 {
		return nil, errors.New("invalid user ID")
	}

	if userID == otherUserID {
		return nil, errors.New(
			"you cannot create a private chat with yourself",
		)
	}

	existing, err := s.conversationRepo.FindPrivateBetween(
		userID,
		otherUserID,
	)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		return existing, nil
	}

	conversation := &models.Conversation{
		ConversationType: "private",
		CreatedBy:        &userID,
	}

	if err := s.conversationRepo.Create(
		conversation,
	); err != nil {
		return nil, err
	}

	// Add the first user.
	firstMember := &models.ConversationMember{
		ConversationID: conversation.ID,
		UserID:         userID,
	}

	if err := s.memberRepo.AddMember(
		firstMember,
	); err != nil {
		return nil, err
	}

	// Add the second user.
	secondMember := &models.ConversationMember{
		ConversationID: conversation.ID,
		UserID:         otherUserID,
	}

	if err := s.memberRepo.AddMember(
		secondMember,
	); err != nil {
		return nil, err
	}

	return conversation, nil
}

// AddMember adds a user to an existing group conversation.
//
// Only an existing member can add another user.
// Private conversations cannot have additional members.
func (s *ChatService) AddMember(
	conversationID,
	requesterID,
	userID int,
) error {
	if conversationID <= 0 {
		return errors.New("invalid conversation ID")
	}

	if requesterID <= 0 || userID <= 0 {
		return errors.New("invalid user ID")
	}

	conversation, err := s.conversationRepo.GetByID(
		conversationID,
	)
	if err != nil {
		return err
	}

	if conversation == nil {
		return errors.New("conversation not found")
	}

	if conversation.ConversationType != "group" {
		return errors.New(
			"members can only be added to groups",
		)
	}

	// Check requester membership.
	isMember, err := s.memberRepo.IsMember(
		conversationID,
		requesterID,
	)
	if err != nil {
		return err
	}

	if !isMember {
		return errors.New(
			"you are not a member of this group",
		)
	}

	// Check whether the target user is already a member.
	alreadyMember, err := s.memberRepo.IsMember(
		conversationID,
		userID,
	)
	if err != nil {
		return err
	}

	if alreadyMember {
		return errors.New(
			"user is already a member of this group",
		)
	}

	member := &models.ConversationMember{
		ConversationID: conversationID,
		UserID:         userID,
	}

	return s.memberRepo.AddMember(member)
}

// RemoveMember removes a member from a group conversation.
//
// Only the group creator can remove members.
// The creator cannot remove themselves through this method.
func (s *ChatService) RemoveMember(
	conversationID,
	requesterID,
	userID int,
) error {
	if conversationID <= 0 {
		return errors.New("invalid conversation ID")
	}

	if requesterID <= 0 || userID <= 0 {
		return errors.New("invalid user ID")
	}

	conversation, err := s.conversationRepo.GetByID(
		conversationID,
	)
	if err != nil {
		return err
	}

	if conversation == nil {
		return errors.New("conversation not found")
	}

	if conversation.ConversationType != "group" {
		return errors.New(
			"members can only be removed from groups",
		)
	}

	// Only the group creator can remove members.
	if conversation.CreatedBy == nil || *conversation.CreatedBy != requesterID {
		return errors.New(
			"only the group creator can remove members",
		)
	}

	// Protect the group creator.
	if conversation.CreatedBy != nil && userID == *conversation.CreatedBy {
		return errors.New(
			"group creator cannot be removed",
		)
	}

	// Verify that the target user is actually a member.
	isMember, err := s.memberRepo.IsMember(
		conversationID,
		userID,
	)
	if err != nil {
		return err
	}

	if !isMember {
		return errors.New(
			"user is not a member of this group",
		)
	}

	return s.memberRepo.RemoveMember(
		conversationID,
		userID,
	)
}

// LeaveGroup allows a normal member to leave a group.
//
// The group creator cannot leave until ownership transfer
// is implemented.
func (s *ChatService) LeaveGroup(
	conversationID,
	userID int,
) error {
	if conversationID <= 0 {
		return errors.New("invalid conversation ID")
	}

	if userID <= 0 {
		return errors.New("invalid user ID")
	}

	conversation, err := s.conversationRepo.GetByID(
		conversationID,
	)
	if err != nil {
		return err
	}

	if conversation == nil {
		return errors.New("conversation not found")
	}

	if conversation.ConversationType != "group" {
		return errors.New("this is not a group")
	}

	if conversation.CreatedBy != nil && *conversation.CreatedBy == userID {
		return errors.New(
			"group creator cannot leave yet",
		)
	}

	isMember, err := s.memberRepo.IsMember(
		conversationID,
		userID,
	)
	if err != nil {
		return err
	}

	if !isMember {
		return errors.New(
			"you are not a member of this group",
		)
	}

	return s.memberRepo.RemoveMember(
		conversationID,
		userID,
	)
}

// SendMessage sends a normal chat message.
//
// Before saving the message, the service verifies that:
//
//  1. The conversation exists.
//  2. The sender is a member.
//  3. The message is not empty.
//  4. The message does not exceed the maximum length.
//
// After saving, the conversation activity timestamp is refreshed
// and other members receive chat notifications when possible.
func (s *ChatService) SendMessage(
	conversationID,
	senderID int,
	message string,
) (*models.ChatMessage, error) {
	if conversationID <= 0 {
		return nil, errors.New("invalid conversation ID")
	}

	if senderID <= 0 {
		return nil, errors.New("invalid sender ID")
	}

	message = strings.TrimSpace(message)

	if message == "" {
		return nil, errors.New(
			"message cannot be empty",
		)
	}

	if len(message) > 5000 {
		return nil, errors.New(
			"message is too long",
		)
	}

	conversation, err := s.conversationRepo.GetByID(
		conversationID,
	)
	if err != nil {
		return nil, err
	}

	if conversation == nil {
		return nil, errors.New(
			"conversation not found",
		)
	}

	isMember, err := s.memberRepo.IsMember(
		conversationID,
		senderID,
	)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New(
			"you are not a member of this conversation",
		)
	}

	chatMessage := &models.ChatMessage{
		ConversationID: conversationID,
		SenderID:       senderID,
		Message:        &message,
		MessageType:    "text",
	}

	if err := s.messageRepo.Create(
		chatMessage,
	); err != nil {
		return nil, err
	}

	// Update conversation activity.
	if err := s.conversationRepo.Touch(
		conversationID,
	); err != nil {
		return nil, err
	}

	// Notify other conversation members.
	//
	// Notification failure does not invalidate the message that
	// was already successfully stored.
	if s.notification != nil {
		if members, memberErr := s.memberRepo.ListMembers(
			conversationID,
		); memberErr == nil {
			for _, member := range members {
				if member.UserID == senderID {
					continue
				}

				_ = s.notification.CreateNotification(
					member.UserID,
					"New chat message",
					"You received a new message in chat.",
					"message",
				)
			}
		}
	}

	return chatMessage, nil
}

// MyConversations returns all conversations belonging to a user.
func (s *ChatService) MyConversations(
	userID int,
) ([]models.Conversation, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user ID")
	}

	return s.conversationRepo.ListForUser(
		userID,
	)
}

// GetConversation retrieves a conversation after verifying
// that the requesting user is a member.
//
// This prevents users from accessing conversations simply by
// changing a conversation ID in a URL or request.
func (s *ChatService) GetConversation(
	conversationID,
	userID int,
) (*models.Conversation, error) {
	if conversationID <= 0 {
		return nil, errors.New("invalid conversation ID")
	}

	if userID <= 0 {
		return nil, errors.New("invalid user ID")
	}

	conversation, err := s.conversationRepo.GetByID(
		conversationID,
	)
	if err != nil {
		return nil, err
	}

	if conversation == nil {
		return nil, errors.New(
			"conversation not found",
		)
	}

	isMember, err := s.memberRepo.IsMember(
		conversationID,
		userID,
	)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New(
			"you are not a member of this conversation",
		)
	}

	return conversation, nil
}

// GetMessages returns messages from a conversation after verifying
// that the requesting user belongs to the conversation.
//
// Successfully loading the messages also marks the conversation
// as read for that user.
func (s *ChatService) GetMessages(
	conversationID,
	userID int,
) ([]models.ChatMessage, error) {
	if conversationID <= 0 {
		return nil, errors.New("invalid conversation ID")
	}

	if userID <= 0 {
		return nil, errors.New("invalid user ID")
	}

	conversation, err := s.conversationRepo.GetByID(
		conversationID,
	)
	if err != nil {
		return nil, err
	}

	if conversation == nil {
		return nil, errors.New(
			"conversation not found",
		)
	}

	isMember, err := s.memberRepo.IsMember(
		conversationID,
		userID,
	)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New(
			"you are not a member of this conversation",
		)
	}

	messages, err := s.messageRepo.ListByConversation(
		conversationID,
	)
	if err != nil {
		return nil, err
	}

	if err := s.memberRepo.MarkRead(conversationID, userID); err != nil {
		return nil, err
	}

	return messages, nil
}

// GetMembers returns the members of a conversation.
//
// The requesting user must already belong to the conversation.
func (s *ChatService) GetMembers(
	conversationID,
	userID int,
) ([]models.ConversationMember, error) {
	if conversationID <= 0 {
		return nil, errors.New("invalid conversation ID")
	}

	if userID <= 0 {
		return nil, errors.New("invalid user ID")
	}

	conversation, err := s.conversationRepo.GetByID(
		conversationID,
	)
	if err != nil {
		return nil, err
	}

	if conversation == nil {
		return nil, errors.New(
			"conversation not found",
		)
	}

	isMember, err := s.memberRepo.IsMember(
		conversationID,
		userID,
	)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, errors.New(
			"you are not a member of this conversation",
		)
	}

	return s.memberRepo.ListMembers(
		conversationID,
	)
}

// DeleteMessage deletes a message belonging to the requesting user.
//
// The message sender is checked before deletion, preventing one user
// from deleting another user's message.
func (s *ChatService) DeleteMessage(
	messageID,
	userID int,
) error {
	if messageID <= 0 {
		return errors.New("invalid message ID")
	}

	if userID <= 0 {
		return errors.New("invalid user ID")
	}

	message, err := s.messageRepo.GetByID(
		messageID,
	)
	if err != nil {
		return err
	}

	if message == nil {
		return errors.New(
			"message not found",
		)
	}

	// Verify that the user belongs to the conversation.
	isMember, err := s.memberRepo.IsMember(
		message.ConversationID,
		userID,
	)
	if err != nil {
		return err
	}

	if !isMember {
		return errors.New(
			"you are not a member of this conversation",
		)
	}

	// Only the sender can delete their own message.
	if message.SenderID != userID {
		return errors.New(
			"you can only delete your own messages",
		)
	}

	return s.messageRepo.Delete(
		messageID,
	)
}

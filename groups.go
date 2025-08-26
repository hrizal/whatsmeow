package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

type JoinGroupRequest struct {
	InviteCode string `json:"invite_code"`
}

type GroupInfoResponse struct {
	JID         string   `json:"jid"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Participants []string `json:"participants"`
	Admins      []string `json:"admins"`
	CreatedAt   int64    `json:"created_at"`
}

type InviteLinkResponse struct {
	InviteLink string `json:"invite_link"`
	InviteCode string `json:"invite_code"`
}

func (api *WhatsAppAPI) createGroup(w http.ResponseWriter, r *http.Request) {
	if !api.client.IsLoggedIn() {
		api.writeError(w, http.StatusUnauthorized, "Not logged in")
		return
	}

	var req CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Parse participant JIDs
	participants := make([]types.JID, len(req.Participants))
	for i, participant := range req.Participants {
		jid, err := types.ParseJID(participant)
		if err != nil {
			api.writeError(w, http.StatusBadRequest, "Invalid participant JID: "+participant)
			return
		}
		participants[i] = jid
	}

	// Create group
	group, err := api.client.CreateGroup(whatsmeow.ReqCreateGroup{
		Name:         req.Name,
		Participants: participants,
	})
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to create group: "+err.Error())
		return
	}

	// Convert participants to strings
	participantStrs := make([]string, len(group.Participants))
	for i, participant := range group.Participants {
		participantStrs[i] = participant.String()
	}

	// Convert admins to strings
	adminStrs := make([]string, len(group.Admins))
	for i, admin := range group.Admins {
		adminStrs[i] = admin.String()
	}

	api.writeSuccess(w, GroupInfoResponse{
		JID:          group.JID.String(),
		Name:         group.Name,
		Description:  group.Description,
		Participants: participantStrs,
		Admins:       adminStrs,
		CreatedAt:    group.CreatedAt.Unix(),
	})
}

func (api *WhatsAppAPI) joinGroup(w http.ResponseWriter, r *http.Request) {
	if !api.client.IsLoggedIn() {
		api.writeError(w, http.StatusUnauthorized, "Not logged in")
		return
	}

	var req JoinGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Extract invite code from invite link if full URL is provided
	inviteCode := req.InviteCode
	if strings.Contains(inviteCode, "chat.whatsapp.com/") {
		parts := strings.Split(inviteCode, "/")
		if len(parts) > 0 {
			inviteCode = parts[len(parts)-1]
		}
	}

	// Join group
	group, err := api.client.JoinGroupWithInvite(inviteCode)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to join group: "+err.Error())
		return
	}

	api.writeSuccess(w, GroupInfoResponse{
		JID:         group.JID.String(),
		Name:        group.Name,
		Description: group.Description,
	})
}

func (api *WhatsAppAPI) leaveGroup(w http.ResponseWriter, r *http.Request) {
	if !api.client.IsLoggedIn() {
		api.writeError(w, http.StatusUnauthorized, "Not logged in")
		return
	}

	var req GroupActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Parse group JID
	groupJID, err := types.ParseJID(req.GroupID)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid group JID: "+err.Error())
		return
	}

	// Leave group
	err = api.client.LeaveGroup(groupJID)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to leave group: "+err.Error())
		return
	}

	api.writeSuccess(w, map[string]interface{}{
		"message": "Successfully left group",
		"group_id": req.GroupID,
	})
}

func (api *WhatsAppAPI) getGroupInfo(w http.ResponseWriter, r *http.Request) {
	if !api.client.IsLoggedIn() {
		api.writeError(w, http.StatusUnauthorized, "Not logged in")
		return
	}

	groupID := r.URL.Query().Get("group_id")
	if groupID == "" {
		api.writeError(w, http.StatusBadRequest, "group_id parameter is required")
		return
	}

	// Parse group JID
	groupJID, err := types.ParseJID(groupID)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid group JID: "+err.Error())
		return
	}

	// Get group info
	group, err := api.client.GetGroupInfo(groupJID)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to get group info: "+err.Error())
		return
	}

	// Convert participants to strings
	participantStrs := make([]string, len(group.Participants))
	for i, participant := range group.Participants {
		participantStrs[i] = participant.String()
	}

	// Convert admins to strings
	adminStrs := make([]string, len(group.Admins))
	for i, admin := range group.Admins {
		adminStrs[i] = admin.String()
	}

	api.writeSuccess(w, GroupInfoResponse{
		JID:          group.JID.String(),
		Name:         group.Name,
		Description:  group.Description,
		Participants: participantStrs,
		Admins:       adminStrs,
		CreatedAt:    group.CreatedAt.Unix(),
	})
}

func (api *WhatsAppAPI) addGroupParticipants(w http.ResponseWriter, r *http.Request) {
	if !api.client.IsLoggedIn() {
		api.writeError(w, http.StatusUnauthorized, "Not logged in")
		return
	}

	var req GroupActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Parse group JID
	groupJID, err := types.ParseJID(req.GroupID)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid group JID: "+err.Error())
		return
	}

	// Parse user JIDs
	users := make([]types.JID, len(req.Users))
	for i, user := range req.Users {
		jid, err := types.ParseJID(user)
		if err != nil {
			api.writeError(w, http.StatusBadRequest, "Invalid user JID: "+user)
			return
		}
		users[i] = jid
	}

	// Add participants
	err = api.client.AddGroupParticipants(groupJID, users)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to add participants: "+err.Error())
		return
	}

	api.writeSuccess(w, map[string]interface{}{
		"message": "Participants added successfully",
		"group_id": req.GroupID,
		"users":    req.Users,
	})
}

func (api *WhatsAppAPI) removeGroupParticipants(w http.ResponseWriter, r *http.Request) {
	if !api.client.IsLoggedIn() {
		api.writeError(w, http.StatusUnauthorized, "Not logged in")
		return
	}

	var req GroupActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Parse group JID
	groupJID, err := types.ParseJID(req.GroupID)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid group JID: "+err.Error())
		return
	}

	// Parse user JIDs
	users := make([]types.JID, len(req.Users))
	for i, user := range req.Users {
		jid, err := types.ParseJID(user)
		if err != nil {
			api.writeError(w, http.StatusBadRequest, "Invalid user JID: "+user)
			return
		}
		users[i] = jid
	}

	// Remove participants
	err = api.client.RemoveGroupParticipants(groupJID, users)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to remove participants: "+err.Error())
		return
	}

	api.writeSuccess(w, map[string]interface{}{
		"message": "Participants removed successfully",
		"group_id": req.GroupID,
		"users":    req.Users,
	})
}

func (api *WhatsAppAPI) getGroupInviteLink(w http.ResponseWriter, r *http.Request) {
	if !api.client.IsLoggedIn() {
		api.writeError(w, http.StatusUnauthorized, "Not logged in")
		return
	}

	groupID := r.URL.Query().Get("group_id")
	if groupID == "" {
		api.writeError(w, http.StatusBadRequest, "group_id parameter is required")
		return
	}

	// Parse group JID
	groupJID, err := types.ParseJID(groupID)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid group JID: "+err.Error())
		return
	}

	// Get invite link
	inviteLink, err := api.client.GetGroupInviteLink(groupJID, false)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to get invite link: "+err.Error())
		return
	}

	// Extract invite code from link
	parts := strings.Split(inviteLink, "/")
	inviteCode := ""
	if len(parts) > 0 {
		inviteCode = parts[len(parts)-1]
	}

	api.writeSuccess(w, InviteLinkResponse{
		InviteLink: inviteLink,
		InviteCode: inviteCode,
	})
}
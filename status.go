package main

import (
	"context"
	"encoding/json"
	"net/http"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
)

type SetStatusRequest struct {
	Status string `json:"status"`
}

type SetPresenceRequest struct {
	Presence string `json:"presence"` // available, unavailable, composing, recording, paused
	To       string `json:"to,omitempty"` // JID to send presence to (optional)
}

func (api *WhatsAppAPI) setStatus(w http.ResponseWriter, r *http.Request) {
	if !api.client.IsLoggedIn() {
		api.writeError(w, http.StatusUnauthorized, "Not logged in")
		return
	}

	var req SetStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Set status
	err := api.client.SetStatusMessage(req.Status)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to set status: "+err.Error())
		return
	}

	api.writeSuccess(w, map[string]interface{}{
		"message": "Status updated successfully",
		"status":  req.Status,
	})
}

func (api *WhatsAppAPI) setPresence(w http.ResponseWriter, r *http.Request) {
	if !api.client.IsLoggedIn() {
		api.writeError(w, http.StatusUnauthorized, "Not logged in")
		return
	}

	var req SetPresenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Parse presence type
	var presence waProto.Presence
	switch req.Presence {
	case "available":
		presence = waProto.Presence_AVAILABLE
	case "unavailable":
		presence = waProto.Presence_UNAVAILABLE
	case "composing":
		presence = waProto.Presence_COMPOSING
	case "recording":
		presence = waProto.Presence_RECORDING
	case "paused":
		presence = waProto.Presence_PAUSED
	default:
		api.writeError(w, http.StatusBadRequest, "Invalid presence type")
		return
	}

	// If 'to' is specified, send presence to specific user
	if req.To != "" {
		recipient, err := types.ParseJID(req.To)
		if err != nil {
			api.writeError(w, http.StatusBadRequest, "Invalid recipient JID: "+err.Error())
			return
		}

		err = api.client.SendPresence(presence, recipient)
		if err != nil {
			api.writeError(w, http.StatusInternalServerError, "Failed to send presence: "+err.Error())
			return
		}
	} else {
		// Send presence to all contacts
		err := api.client.SendPresence(presence, types.EmptyJID)
		if err != nil {
			api.writeError(w, http.StatusInternalServerError, "Failed to send presence: "+err.Error())
			return
		}
	}

	api.writeSuccess(w, map[string]interface{}{
		"message":  "Presence updated successfully",
		"presence": req.Presence,
		"to":       req.To,
	})
}
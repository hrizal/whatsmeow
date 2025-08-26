package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
)

type DeleteMessageRequest struct {
	ChatJID    string `json:"chat_jid"`
	MessageID  string `json:"message_id"`
	ForEveryone bool  `json:"for_everyone"`
}

type SendMediaRequest struct {
	To      string `json:"to"`
	Type    string `json:"type"` // image, video, audio, document
	Caption string `json:"caption,omitempty"`
	URL     string `json:"url,omitempty"`
}

type MessageResponse struct {
	MessageID string `json:"message_id"`
	ChatJID   string `json:"chat_jid"`
	Timestamp int64  `json:"timestamp"`
}

func (api *WhatsAppAPI) sendMessage(w http.ResponseWriter, r *http.Request) {
	if !api.client.IsLoggedIn() {
		api.writeError(w, http.StatusUnauthorized, "Not logged in")
		return
	}

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Parse JID
	recipient, err := types.ParseJID(req.To)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid recipient JID: "+err.Error())
		return
	}

	// Create message
	msg := &waProto.Message{
		Conversation: &req.Message,
	}

	// Send message
	resp, err := api.client.SendMessage(context.Background(), recipient, msg)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to send message: "+err.Error())
		return
	}

	api.writeSuccess(w, MessageResponse{
		MessageID: resp.ID,
		ChatJID:   recipient.String(),
		Timestamp: resp.Timestamp.Unix(),
	})
}

func (api *WhatsAppAPI) sendMedia(w http.ResponseWriter, r *http.Request) {
	if !api.client.IsLoggedIn() {
		api.writeError(w, http.StatusUnauthorized, "Not logged in")
		return
	}

	var req SendMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Parse JID
	recipient, err := types.ParseJID(req.To)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid recipient JID: "+err.Error())
		return
	}

	// Download media from URL
	resp, err := http.Get(req.URL)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Failed to download media: "+err.Error())
		return
	}
	defer resp.Body.Close()

	mediaData, err := io.ReadAll(resp.Body)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to read media data")
		return
	}

	// Upload media
	uploaded, err := api.client.Upload(context.Background(), mediaData, whatsmeow.MediaImage)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to upload media: "+err.Error())
		return
	}

	// Create media message based on type
	var msg *waProto.Message
	switch strings.ToLower(req.Type) {
	case "image":
		msg = &waProto.Message{
			ImageMessage: &waProto.ImageMessage{
				Url:           &uploaded.URL,
				Mimetype:      &uploaded.Mimetype,
				Caption:       &req.Caption,
				FileLength:    &uploaded.FileLength,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSha256: uploaded.FileEncSHA256,
				FileSha256:    uploaded.FileSHA256,
			},
		}
	case "video":
		msg = &waProto.Message{
			VideoMessage: &waProto.VideoMessage{
				Url:           &uploaded.URL,
				Mimetype:      &uploaded.Mimetype,
				Caption:       &req.Caption,
				FileLength:    &uploaded.FileLength,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSha256: uploaded.FileEncSHA256,
				FileSha256:    uploaded.FileSHA256,
			},
		}
	case "audio":
		msg = &waProto.Message{
			AudioMessage: &waProto.AudioMessage{
				Url:           &uploaded.URL,
				Mimetype:      &uploaded.Mimetype,
				FileLength:    &uploaded.FileLength,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSha256: uploaded.FileEncSHA256,
				FileSha256:    uploaded.FileSHA256,
			},
		}
	case "document":
		msg = &waProto.Message{
			DocumentMessage: &waProto.DocumentMessage{
				Url:           &uploaded.URL,
				Mimetype:      &uploaded.Mimetype,
				FileName:      &req.Caption,
				FileLength:    &uploaded.FileLength,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSha256: uploaded.FileEncSHA256,
				FileSha256:    uploaded.FileSHA256,
			},
		}
	default:
		api.writeError(w, http.StatusBadRequest, "Unsupported media type")
		return
	}

	// Send message
	sendResp, err := api.client.SendMessage(context.Background(), recipient, msg)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to send media message: "+err.Error())
		return
	}

	api.writeSuccess(w, MessageResponse{
		MessageID: sendResp.ID,
		ChatJID:   recipient.String(),
		Timestamp: sendResp.Timestamp.Unix(),
	})
}

func (api *WhatsAppAPI) deleteMessage(w http.ResponseWriter, r *http.Request) {
	if !api.client.IsLoggedIn() {
		api.writeError(w, http.StatusUnauthorized, "Not logged in")
		return
	}

	var req DeleteMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Parse JID
	chatJID, err := types.ParseJID(req.ChatJID)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid chat JID: "+err.Error())
		return
	}

	// Delete message
	err = api.client.DeleteMessage(chatJID, req.MessageID, req.ForEveryone)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to delete message: "+err.Error())
		return
	}

	api.writeSuccess(w, map[string]interface{}{
		"message": "Message deleted successfully",
		"message_id": req.MessageID,
		"for_everyone": req.ForEveryone,
	})
}
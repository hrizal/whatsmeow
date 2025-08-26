package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

type UploadMediaResponse struct {
	URL           string `json:"url"`
	DirectPath    string `json:"direct_path"`
	MediaKey      []byte `json:"media_key"`
	FileEncSHA256 []byte `json:"file_enc_sha256"`
	FileSHA256    []byte `json:"file_sha256"`
	FileLength    uint64 `json:"file_length"`
	Mimetype      string `json:"mimetype"`
}

func (api *WhatsAppAPI) uploadMedia(w http.ResponseWriter, r *http.Request) {
	if !api.client.IsLoggedIn() {
		api.writeError(w, http.StatusUnauthorized, "Not logged in")
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(32 << 20) // 32MB max
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Failed to parse form: "+err.Error())
		return
	}

	// Get file from form
	file, header, err := r.FormFile("file")
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "No file provided: "+err.Error())
		return
	}
	defer file.Close()

	// Read file data
	fileData, err := io.ReadAll(file)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to read file: "+err.Error())
		return
	}

	// Determine media type based on file extension
	ext := filepath.Ext(header.Filename)
	var mediaType whatsmeow.MediaType
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		mediaType = whatsmeow.MediaImage
	case ".mp4", ".avi", ".mov", ".mkv":
		mediaType = whatsmeow.MediaVideo
	case ".mp3", ".wav", ".ogg", ".m4a":
		mediaType = whatsmeow.MediaAudio
	case ".pdf", ".doc", ".docx", ".txt", ".zip", ".rar":
		mediaType = whatsmeow.MediaDocument
	default:
		mediaType = whatsmeow.MediaDocument
	}

	// Upload media
	uploaded, err := api.client.Upload(context.Background(), fileData, mediaType)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to upload media: "+err.Error())
		return
	}

	api.writeSuccess(w, UploadMediaResponse{
		URL:           uploaded.URL,
		DirectPath:    uploaded.DirectPath,
		MediaKey:      uploaded.MediaKey,
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    uploaded.FileLength,
		Mimetype:      uploaded.Mimetype,
	})
}

func (api *WhatsAppAPI) downloadMedia(w http.ResponseWriter, r *http.Request) {
	if !api.client.IsLoggedIn() {
		api.writeError(w, http.StatusUnauthorized, "Not logged in")
		return
	}

	vars := mux.Vars(r)
	messageID := vars["messageID"]

	// Get message info from store (this is a simplified approach)
	// In a real implementation, you'd need to store message metadata
	// For now, we'll return an error indicating this needs message context
	api.writeError(w, http.StatusNotImplemented, "Download media requires message context. Please provide message details in request body.")

	// Example of how it would work with message context:
	/*
	// Parse request body for message details
	var req struct {
		ChatJID   string `json:"chat_jid"`
		MessageID string `json:"message_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Parse chat JID
	chatJID, err := types.ParseJID(req.ChatJID)
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "Invalid chat JID: "+err.Error())
		return
	}

	// Download media
	data, err := api.client.Download(context.Background(), &types.MessageInfo{
		Chat:     chatJID,
		ID:       req.MessageID,
		// Other message fields would be needed
	})
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to download media: "+err.Error())
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=media_%s", req.MessageID))
	
	// Write file data
	w.Write(data)
	*/
}
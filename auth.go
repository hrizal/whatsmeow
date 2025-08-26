package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type QRCodeResponse struct {
	QRCode string `json:"qr_code"`
	ExpiresAt time.Time `json:"expires_at"`
}

type AuthStatusResponse struct {
	IsLoggedIn bool   `json:"is_logged_in"`
	UserID     string `json:"user_id,omitempty"`
	Platform   string `json:"platform,omitempty"`
}

func (api *WhatsAppAPI) healthCheck(w http.ResponseWriter, r *http.Request) {
	api.writeSuccess(w, map[string]interface{}{
		"status": "healthy",
		"time":   time.Now().UTC(),
	})
}

func (api *WhatsAppAPI) getAuthStatus(w http.ResponseWriter, r *http.Request) {
	if !api.client.IsLoggedIn() {
		api.writeSuccess(w, AuthStatusResponse{
			IsLoggedIn: false,
		})
		return
	}

	userID := api.client.Store.ID
	platform := api.client.Store.Platform

	api.writeSuccess(w, AuthStatusResponse{
		IsLoggedIn: true,
		UserID:     userID.String(),
		Platform:   platform,
	})
}

func (api *WhatsAppAPI) getQRCode(w http.ResponseWriter, r *http.Request) {
	if api.client.IsLoggedIn() {
		api.writeError(w, http.StatusBadRequest, "Already logged in")
		return
	}

	// Start QR code generation
	qrChan, _ := api.client.GetQRChannel(context.Background())
	err := api.client.Connect()
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to connect: "+err.Error())
		return
	}

	// Wait for QR code
	select {
	case evt := <-qrChan:
		if evt.Event == "code" {
			api.writeSuccess(w, QRCodeResponse{
				QRCode:   evt.Code,
				ExpiresAt: time.Now().Add(2 * time.Minute), // QR codes typically expire in 2 minutes
			})
		} else {
			api.writeError(w, http.StatusInternalServerError, "Failed to generate QR code")
		}
	case <-time.After(30 * time.Second):
		api.writeError(w, http.StatusRequestTimeout, "QR code generation timeout")
	}
}

func (api *WhatsAppAPI) logout(w http.ResponseWriter, r *http.Request) {
	if !api.client.IsLoggedIn() {
		api.writeError(w, http.StatusBadRequest, "Not logged in")
		return
	}

	err := api.client.Logout()
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "Failed to logout: "+err.Error())
		return
	}

	api.writeSuccess(w, map[string]interface{}{
		"message": "Successfully logged out",
	})
}
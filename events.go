package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

type EventData struct {
	Type      string      `json:"type"`
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data"`
}

func (api *WhatsAppAPI) eventStream(w http.ResponseWriter, r *http.Request) {
	if !api.client.IsLoggedIn() {
		api.writeError(w, http.StatusUnauthorized, "Not logged in")
		return
	}

	// Set headers for Server-Sent Events
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Create a channel to handle client disconnect
	notify := w.(http.CloseNotifier).CloseNotify()

	// Set up event handler
	api.client.AddEventHandler(func(evt interface{}) {
		select {
		case <-notify:
			// Client disconnected
			return
		default:
			// Process event
			eventData := api.processEvent(evt)
			if eventData != nil {
				// Send event as SSE
				data, _ := json.Marshal(eventData)
				fmt.Fprintf(w, "data: %s\n\n", data)
				w.(http.Flusher).Flush()
			}
		}
	})

	// Keep connection alive
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-notify:
			// Client disconnected
			return
		case <-ticker.C:
			// Send keepalive
			fmt.Fprintf(w, ": keepalive\n\n")
			w.(http.Flusher).Flush()
		}
	}
}

func (api *WhatsAppAPI) processEvent(evt interface{}) *EventData {
	switch e := evt.(type) {
	case *events.Message:
		return &EventData{
			Type:      "message",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"id":        e.Info.ID,
				"chat_jid":  e.Info.Chat.String(),
				"sender_jid": e.Info.Sender.String(),
				"timestamp": e.Info.Timestamp.Unix(),
				"type":      e.Message.GetConversation(),
				"push_name": e.Info.PushName,
			},
		}
	case *events.Receipt:
		return &EventData{
			Type:      "receipt",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"message_ids": e.MessageIDs,
				"sender_jid":  e.Sender.String(),
				"type":        e.Type,
				"timestamp":   e.Timestamp.Unix(),
			},
		}
	case *events.Presence:
		return &EventData{
			Type:      "presence",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"user_jid": e.From.String(),
				"presence": e.Presence,
				"last_seen": e.LastSeen.Unix(),
			},
		}
	case *events.JoinedGroup:
		return &EventData{
			Type:      "joined_group",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"group_jid": e.JID.String(),
				"name":      e.Name,
				"create_key": e.CreateKey,
			},
		}
	case *events.LeftGroup:
		return &EventData{
			Type:      "left_group",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"group_jid": e.JID.String(),
				"name":      e.Name,
			},
		}
	case *events.GroupParticipants:
		return &EventData{
			Type:      "group_participants",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"group_jid":   e.JID.String(),
				"participants": e.Participants,
				"action":      e.Action,
				"actor":       e.Actor.String(),
			},
		}
	case *events.Contact:
		return &EventData{
			Type:      "contact",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"jid":         e.JID.String(),
				"first_name":  e.FirstName,
				"full_name":   e.FullName,
				"push_name":   e.PushName,
				"business_name": e.BusinessName,
			},
		}
	case *events.Connected:
		return &EventData{
			Type:      "connected",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"message": "WhatsApp client connected",
			},
		}
	case *events.Disconnected:
		return &EventData{
			Type:      "disconnected",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"message": "WhatsApp client disconnected",
				"reason":  e.Reason,
			},
		}
	default:
		// Unknown event type
		return &EventData{
			Type:      "unknown",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"message": fmt.Sprintf("Unknown event type: %T", evt),
			},
		}
	}
}
# WhatsApp API Server

A comprehensive REST API server built with Go that exposes the functionality of the [whatsmeow](https://github.com/tulir/whatsmeow) WhatsApp library. This API allows you to interact with WhatsApp Web through HTTP endpoints.

## Features

- **Authentication**: QR code generation and login management
- **Messaging**: Send text and media messages
- **Group Management**: Create, join, leave groups, manage participants
- **Contact Management**: Get contact information and lists
- **Media Operations**: Upload and download media files
- **Status & Presence**: Set status messages and presence
- **Real-time Events**: Server-Sent Events (SSE) for real-time updates
- **Database Storage**: SQLite-based message and contact storage

## Prerequisites

- Go 1.21 or higher
- SQLite3
- WhatsApp account for authentication

## Installation

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd whatsapp-api
   ```

2. **Install dependencies**:
   ```bash
   go mod tidy
   ```

3. **Run the server**:
   ```bash
   go run *.go
   ```

The server will start on port 8080 by default. You can change this by setting the `PORT` environment variable.

## API Endpoints

### Authentication

#### GET /health
Health check endpoint.

**Response**:
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "time": "2024-01-01T12:00:00Z"
  }
}
```

#### GET /auth/status
Check authentication status.

**Response**:
```json
{
  "success": true,
  "data": {
    "is_logged_in": true,
    "user_id": "1234567890@s.whatsapp.net",
    "platform": "Chrome"
  }
}
```

#### GET /auth/qr
Generate QR code for WhatsApp Web authentication.

**Response**:
```json
{
  "success": true,
  "data": {
    "qr_code": "2@...",
    "expires_at": "2024-01-01T12:02:00Z"
  }
}
```

#### POST /auth/logout
Logout from WhatsApp.

**Response**:
```json
{
  "success": true,
  "data": {
    "message": "Successfully logged out"
  }
}
```

### Messaging

#### POST /messages/send
Send a text message.

**Request Body**:
```json
{
  "to": "1234567890@s.whatsapp.net",
  "message": "Hello, World!",
  "type": "text"
}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "message_id": "3EB0...",
    "chat_jid": "1234567890@s.whatsapp.net",
    "timestamp": 1704110400
  }
}
```

#### POST /messages/send-media
Send a media message.

**Request Body**:
```json
{
  "to": "1234567890@s.whatsapp.net",
  "type": "image",
  "caption": "Check this out!",
  "url": "https://example.com/image.jpg"
}
```

**Supported media types**: `image`, `video`, `audio`, `document`

#### DELETE /messages/delete
Delete a message.

**Request Body**:
```json
{
  "chat_jid": "1234567890@s.whatsapp.net",
  "message_id": "3EB0...",
  "for_everyone": true
}
```

### Group Management

#### POST /groups/create
Create a new group.

**Request Body**:
```json
{
  "name": "My Group",
  "participants": [
    "1234567890@s.whatsapp.net",
    "0987654321@s.whatsapp.net"
  ]
}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "jid": "1234567890-1234567890@g.us",
    "name": "My Group",
    "description": "",
    "participants": [
      "1234567890@s.whatsapp.net",
      "0987654321@s.whatsapp.net"
    ],
    "admins": ["1234567890@s.whatsapp.net"],
    "created_at": 1704110400
  }
}
```

#### POST /groups/join
Join a group using invite code.

**Request Body**:
```json
{
  "invite_code": "ABC123"
}
```

#### POST /groups/leave
Leave a group.

**Request Body**:
```json
{
  "group_id": "1234567890-1234567890@g.us"
}
```

#### GET /groups/info?group_id=1234567890-1234567890@g.us
Get group information.

#### POST /groups/participants/add
Add participants to a group.

**Request Body**:
```json
{
  "group_id": "1234567890-1234567890@g.us",
  "users": [
    "1234567890@s.whatsapp.net",
    "0987654321@s.whatsapp.net"
  ]
}
```

#### POST /groups/participants/remove
Remove participants from a group.

**Request Body**:
```json
{
  "group_id": "1234567890-1234567890@g.us",
  "users": [
    "1234567890@s.whatsapp.net"
  ]
}
```

#### GET /groups/invite-link?group_id=1234567890-1234567890@g.us
Get group invite link.

**Response**:
```json
{
  "success": true,
  "data": {
    "invite_link": "https://chat.whatsapp.com/ABC123",
    "invite_code": "ABC123"
  }
}
```

### Contact Management

#### GET /contacts
Get all contacts.

**Response**:
```json
{
  "success": true,
  "data": [
    {
      "jid": "1234567890@s.whatsapp.net",
      "name": "John",
      "full_name": "John Doe",
      "push_name": "John",
      "business_name": "",
      "verified_name": ""
    }
  ]
}
```

#### GET /contacts/{jid}
Get specific contact information.

**Response**:
```json
{
  "success": true,
  "data": {
    "jid": "1234567890@s.whatsapp.net",
    "name": "John",
    "full_name": "John Doe",
    "push_name": "John",
    "business_name": "",
    "verified_name": ""
  }
}
```

### Media Operations

#### POST /media/upload
Upload a media file.

**Request**: Multipart form with `file` field.

**Response**:
```json
{
  "success": true,
  "data": {
    "url": "https://mmg.whatsapp.net/...",
    "direct_path": "/v/t62.7118-24/...",
    "media_key": "...",
    "file_enc_sha256": "...",
    "file_sha256": "...",
    "file_length": 12345,
    "mimetype": "image/jpeg"
  }
}
```

#### GET /media/download/{messageID}
Download media from a message (requires message context).

### Status & Presence

#### POST /status/set
Set status message.

**Request Body**:
```json
{
  "status": "Available for work"
}
```

#### POST /presence/set
Set presence status.

**Request Body**:
```json
{
  "presence": "available",
  "to": "1234567890@s.whatsapp.net"
}
```

**Supported presence types**: `available`, `unavailable`, `composing`, `recording`, `paused`

### Real-time Events

#### GET /events
Stream real-time WhatsApp events using Server-Sent Events (SSE).

**Event Types**:
- `message`: New message received
- `receipt`: Message delivery/read receipts
- `presence`: User presence updates
- `joined_group`: Joined a group
- `left_group`: Left a group
- `group_participants`: Group participant changes
- `contact`: Contact updates
- `connected`: Client connected
- `disconnected`: Client disconnected

**Example Event**:
```
data: {"type":"message","timestamp":1704110400,"data":{"id":"3EB0...","chat_jid":"1234567890@s.whatsapp.net","sender_jid":"1234567890@s.whatsapp.net","timestamp":1704110400,"type":"Hello!","push_name":"John"}}
```

## Error Handling

All endpoints return consistent error responses:

```json
{
  "success": false,
  "error": "Error message description"
}
```

Common HTTP status codes:
- `200`: Success
- `400`: Bad Request (invalid parameters)
- `401`: Unauthorized (not logged in)
- `404`: Not Found
- `500`: Internal Server Error

## JID Format

WhatsApp uses JID (Jabber ID) format for identifying users and groups:

- **User JID**: `1234567890@s.whatsapp.net`
- **Group JID**: `1234567890-1234567890@g.us`
- **Broadcast JID**: `1234567890@broadcast`

## Security Considerations

- The API currently has no authentication mechanism - implement proper authentication for production use
- Store sensitive data securely
- Use HTTPS in production
- Implement rate limiting
- Validate all input data

## Development

### Project Structure

```
whatsapp-api/
├── main.go          # Main application entry point
├── auth.go          # Authentication handlers
├── messages.go      # Message handlers
├── groups.go        # Group management handlers
├── contacts.go      # Contact management handlers
├── media.go         # Media upload/download handlers
├── status.go        # Status and presence handlers
├── events.go        # Real-time event streaming
├── api_go.mod       # Go module file
├── API_README.md    # This documentation
└── whatsapp.db      # SQLite database (created automatically)
```

### Adding New Endpoints

1. Add the route in `main.go` `setupRoutes()` function
2. Create the handler function in the appropriate file
3. Follow the existing pattern for request/response handling
4. Add documentation to this README

### Testing

```bash
# Test health endpoint
curl http://localhost:8080/health

# Test authentication status
curl http://localhost:8080/auth/status

# Test sending a message (after authentication)
curl -X POST http://localhost:8080/messages/send \
  -H "Content-Type: application/json" \
  -d '{"to":"1234567890@s.whatsapp.net","message":"Hello!"}'
```

## Troubleshooting

### Common Issues

1. **QR Code not generating**: Make sure the client is not already logged in
2. **Message sending fails**: Verify the recipient JID format
3. **Group operations fail**: Ensure you have the necessary permissions
4. **Media upload fails**: Check file size and format

### Logs

The application logs all requests and errors. Check the console output for debugging information.

## License

This project is licensed under the same license as the whatsmeow library (Mozilla Public License 2.0).

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## Support

For issues related to:
- **WhatsApp protocol**: Check [whatsmeow discussions](https://github.com/tulir/whatsmeow/discussions)
- **This API**: Open an issue in this repository
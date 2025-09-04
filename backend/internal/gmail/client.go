package gmail

import (
	"context"
	"encoding/base64"
	"fmt"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
	"golang.org/x/oauth2"
)

type Client struct {
	service *gmail.Service
}

type AttachmentRequest struct {
	MessageID    string `json:"messageId"`
	AttachmentID string `json:"attachmentId"`
}

func NewClient(ctx context.Context, token *oauth2.Token) (*Client, error) {
	service, err := gmail.NewService(ctx, option.WithTokenSource(oauth2.StaticTokenSource(token)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %v", err)
	}

	return &Client{service: service}, nil
}

func (c *Client) GetAttachment(ctx context.Context, messageID, attachmentID string) ([]byte, string, error) {
	// Get the message to verify access
	msg, err := c.service.Users.Messages.Get("me", messageID).Do()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get message: %v", err)
	}

	// Find the attachment
	var attachment *gmail.MessagePartBody
	var filename string
	var mimeType string

	// Search through message parts for the attachment
	err = c.findAttachment(msg.Payload, attachmentID, &attachment, &filename, &mimeType)
	if err != nil {
		return nil, "", err
	}

	if attachment == nil {
		return nil, "", fmt.Errorf("attachment not found")
	}

	// Get attachment data
	attachmentData, err := c.service.Users.Messages.Attachments.Get("me", messageID, attachment.AttachmentId).Do()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get attachment data: %v", err)
	}

	// Decode base64 data
	data, err := base64.URLEncoding.DecodeString(attachmentData.Data)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode attachment data: %v", err)
	}

	return data, mimeType, nil
}

func (c *Client) findAttachment(part *gmail.MessagePart, attachmentID string, result **gmail.MessagePartBody, filename *string, mimeType *string) error {
	// Check if this part is the attachment we're looking for
	if part.Body != nil && part.Body.AttachmentId == attachmentID {
		*result = part.Body
		*mimeType = part.MimeType
		
		// Try to get filename from headers
		if part.Filename != "" {
			*filename = part.Filename
		}
		return nil
	}

	// Search in parts recursively
	for _, subPart := range part.Parts {
		err := c.findAttachment(subPart, attachmentID, result, filename, mimeType)
		if err == nil && *result != nil {
			return nil
		}
	}

	return nil
}

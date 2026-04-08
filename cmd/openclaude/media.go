package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sdk "github.com/sashabaranov/go-openai"
)

const maxImageFileBytes = 8 << 20 // 8 MiB

// buildUserContentParts returns OpenAI-style multimodal parts: text first, then image_url entries.
func buildUserContentParts(text string, imageURLs []string, imageFiles []string) ([]sdk.ChatMessagePart, error) {
	text = strings.TrimSpace(text)
	var parts []sdk.ChatMessagePart
	if text != "" {
		parts = append(parts, sdk.ChatMessagePart{
			Type: sdk.ChatMessagePartTypeText,
			Text: text,
		})
	}
	for _, u := range imageURLs {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		parts = append(parts, sdk.ChatMessagePart{
			Type: sdk.ChatMessagePartTypeImageURL,
			ImageURL: &sdk.ChatMessageImageURL{
				URL: u,
			},
		})
	}
	for _, fp := range imageFiles {
		fp = strings.TrimSpace(fp)
		if fp == "" {
			continue
		}
		dataURL, err := imageFileDataURL(fp)
		if err != nil {
			return nil, err
		}
		parts = append(parts, sdk.ChatMessagePart{
			Type: sdk.ChatMessagePartTypeImageURL,
			ImageURL: &sdk.ChatMessageImageURL{
				URL: dataURL,
			},
		})
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("no user text or images")
	}
	return parts, nil
}

func imageFileDataURL(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	if len(b) > maxImageFileBytes {
		return "", fmt.Errorf("image file too large (%d bytes; max %d)", len(b), maxImageFileBytes)
	}
	mime := mimeForImagePath(abs)
	enc := base64.StdEncoding.EncodeToString(b)
	return fmt.Sprintf("data:%s;base64,%s", mime, enc), nil
}

func mimeForImagePath(p string) string {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	default:
		return "application/octet-stream"
	}
}

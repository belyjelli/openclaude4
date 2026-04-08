package core

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sdk "github.com/sashabaranov/go-openai"
)

const maxImageFileBytes = 8 << 20 // 8 MiB

// MaxGRPCImageAttachments caps image_url + image_inline entries on gRPC ChatRequest (per turn).
const MaxGRPCImageAttachments = 16

// GRPCInlineImage is raw image bytes + MIME type for gRPC multimodal turns.
type GRPCInlineImage struct {
	Data []byte
	MIME string
}

// BuildUserContentPartsFromGRPC builds multimodal parts from gRPC-style URLs and inline attachments.
// Enforces [MaxGRPCImageAttachments] total images and [maxImageFileBytes] per inline blob.
func BuildUserContentPartsFromGRPC(userText string, imageURLs []string, inlines []GRPCInlineImage) ([]sdk.ChatMessagePart, error) {
	var urls []string
	for _, u := range imageURLs {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		urls = append(urls, u)
	}
	nImg := len(urls) + len(inlines)
	if nImg > MaxGRPCImageAttachments {
		return nil, fmt.Errorf("too many images (%d; max %d)", nImg, MaxGRPCImageAttachments)
	}
	for i := range inlines {
		in := inlines[i]
		if len(in.Data) == 0 {
			return nil, fmt.Errorf("image_inline[%d]: empty data", i)
		}
		if len(in.Data) > maxImageFileBytes {
			return nil, fmt.Errorf("image_inline[%d]: data too large (%d bytes; max %d)", i, len(in.Data), maxImageFileBytes)
		}
		mime := strings.TrimSpace(in.MIME)
		if mime == "" {
			return nil, fmt.Errorf("image_inline[%d]: empty mime_type", i)
		}
		enc := base64.StdEncoding.EncodeToString(in.Data)
		urls = append(urls, fmt.Sprintf("data:%s;base64,%s", mime, enc))
	}
	return BuildUserContentParts(userText, urls, nil)
}

// BuildUserContentParts builds OpenAI-style multimodal parts: text first, then image_url entries.
// Empty text with images gets a short placeholder so the API always sees a text part.
func BuildUserContentParts(text string, imageURLs []string, imageFiles []string) ([]sdk.ChatMessagePart, error) {
	text = strings.TrimSpace(text)
	hasImg := false
	for _, u := range imageURLs {
		if strings.TrimSpace(u) != "" {
			hasImg = true
			break
		}
	}
	if !hasImg {
		for _, f := range imageFiles {
			if strings.TrimSpace(f) != "" {
				hasImg = true
				break
			}
		}
	}
	if text == "" && hasImg {
		text = "Answer based on the attached image(s)."
	}
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

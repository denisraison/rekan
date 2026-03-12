package whatsapp

import (
	"context"

	"go.mau.fi/whatsmeow/types/events"

	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func transcribeAudio(ctx context.Context, deps HandlerDeps, evt *events.Message) string {
	if deps.Transcribe == nil {
		deps.Logger.Warn("whatsapp: audio received but no transcription client configured")
		return ""
	}

	audio := evt.Message.GetAudioMessage()
	if audio == nil {
		return ""
	}

	data, err := deps.Client.Download(ctx, audio)
	if err != nil {
		deps.Logger.Error("whatsapp: failed to download audio", "error", err)
		return ""
	}

	text, err := deps.Transcribe.Transcribe(ctx, data, "audio/ogg")
	if err != nil {
		deps.Logger.Error("whatsapp: transcription failed", "error", err)
		return ""
	}

	return text
}

func processVideo(ctx context.Context, deps HandlerDeps, evt *events.Message) (description, caption string, file *filesystem.File) {
	vid := evt.Message.GetVideoMessage()
	if vid == nil {
		return "", "", nil
	}

	data, err := deps.Client.Download(ctx, vid)
	if err != nil {
		deps.Logger.Error("whatsapp: failed to download video", "error", err)
		return "", "", nil
	}

	mimeType := vid.GetMimetype()
	ext := ".mp4"
	if mimeType == "video/3gpp" {
		ext = ".3gp"
	}

	filename := string(evt.Info.ID) + ext
	f, err := filesystem.NewFileFromBytes(data, filename)
	if err != nil {
		deps.Logger.Error("whatsapp: failed to create file from bytes", "error", err)
		return "", "", nil
	}

	caption = vid.GetCaption()
	description = caption // fallback if Gemini is unavailable

	if deps.Transcribe != nil {
		desc, err := deps.Transcribe.DescribeVideo(ctx, data, mimeType, caption)
		if err != nil {
			deps.Logger.Error("whatsapp: failed to describe video", "error", err)
		} else {
			description = desc
		}
	}

	return description, caption, f
}

func processImage(ctx context.Context, deps HandlerDeps, evt *events.Message) (description, caption string, file *filesystem.File) {
	img := evt.Message.GetImageMessage()
	if img == nil {
		return "", "", nil
	}

	data, err := deps.Client.Download(ctx, img)
	if err != nil {
		deps.Logger.Error("whatsapp: failed to download image", "error", err)
		return "", "", nil
	}

	mimeType := img.GetMimetype()
	ext := ".jpg"
	switch mimeType {
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	}

	filename := string(evt.Info.ID) + ext
	f, err := filesystem.NewFileFromBytes(data, filename)
	if err != nil {
		deps.Logger.Error("whatsapp: failed to create file from bytes", "error", err)
		return "", "", nil
	}

	caption = img.GetCaption()
	description = caption // fallback if Gemini is unavailable

	if deps.Transcribe != nil {
		desc, err := deps.Transcribe.DescribeImage(ctx, data, mimeType, caption)
		if err != nil {
			deps.Logger.Error("whatsapp: failed to describe image", "error", err)
		} else {
			description = desc
		}
	}

	return description, caption, f
}

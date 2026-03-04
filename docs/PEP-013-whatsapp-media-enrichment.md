# PEP-013: WhatsApp Media Enrichment

**Status:** Done
**Date:** 2026-03-04

## Context

When clients send images or videos via WhatsApp, the operator previously saw little to no useful context:

- **Images**: the raw image was stored and shown in the chat UI, but only the caption (if any) was captured as text. Gemini was never called to describe the image.
- **Videos**: completely ignored. The handler dropped video messages on the floor — no storage, no display, no processing.
- **Profile pictures**: WhatsApp pushName was captured as `client_name`, but no profile photo was fetched. The chat header and client info screen showed no avatar.

Gemini was already integrated for audio transcription (`api/internal/transcribe/gemini.go`). All three gaps are now closed.

## Waves

### Wave 1: Gemini image description ✓

- `api/internal/transcribe/gemini.go`: extracted shared `call` helper; added `DescribeImage(ctx, imageBytes, mimeType, caption)`. Caption is included as a hint in the prompt when present.
- `api/internal/whatsapp/handler.go`: replaced `downloadImage` with `processImage` returning `(description, caption, file)`. Gemini description stored as `content`; raw caption saved as a separate text message (see caption fix below).
- `api/internal/transcribe/gemini_test.go`: `TestDescribeImage` covers description returned, caption in prompt, and error on non-200.

**Note added during implementation:** when an image or video carries a caption, we now save it as an additional `text` message (with `wa_message_id + "_caption"` for deduplication) so the operator sees both the AI description and the client's own words.

### Wave 2: Video support ✓

- `api/migrations/1740000020_messages_add_video_type.go`: adds `video` to `messages.type` select; raises `media` file size limit from 10 MB to 50 MB.
- `api/internal/domain/domain.go`: added `MsgTypeVideo = "video"`.
- `api/internal/transcribe/gemini.go`: added `DescribeVideo(ctx, videoBytes, mimeType, caption)`.
- `api/internal/whatsapp/handler.go`: added `processVideo` (same shape as `processImage`) and wired `MsgTypeVideo` into the switch.
- `web/src/lib/types.ts`: added `'video'` to `Message.type` union.
- `web/src/routes/(app)/operador/+page.svelte`: added `<video controls>` block alongside the image block.
- `api/internal/transcribe/gemini_test.go`: `TestDescribeVideo` covers the same cases as `TestDescribeImage`.

### Wave 3: Client profile picture ✓

- `api/migrations/1740000021_businesses_profile_picture.go`: adds `profile_picture` (FileField, 2 MB), `profile_picture_id`, and `profile_picture_updated` to `businesses`.
- `api/internal/whatsapp/client.go`: added `GetProfilePicture` (wraps `wac.GetProfilePictureInfo`) and `DownloadURL` (plain HTTP GET).
- `api/internal/whatsapp/handler.go`: after every incoming message, spawns `refreshProfilePicture` in a goroutine. Skips if updated within 7 days; handles not-set/unauthorized silently; stamps `profile_picture_updated` even on failure to avoid hammering the API.
- `web/src/lib/types.ts`: added `collectionId` and `profile_picture` to `Business`.
- `web/src/routes/(app)/operador/+page.svelte`: added `profilePictureUrl` and `initials` helpers; circular avatar with initials fallback in both the chat header and the info screen header.

## Consequences

- Operators see a Gemini-generated Portuguese description for every image and video, plus the client's original caption as a separate message when provided.
- Video messages are no longer silently dropped.
- The `messages` type enum gains `video`; additive and backward-compatible.
- Gemini API usage increases by one call per incoming image or video.
- Profile pictures are fetched from WhatsApp on first contact and refreshed at most once every 7 days per business.
- The 50 MB media file size cap for videos is a deliberate choice; clips larger than that are rejected by whatsmeow's download layer.

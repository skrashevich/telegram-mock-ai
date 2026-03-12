package api

import (
	"bytes"
	"crypto/sha256"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/skrashevich/go-webp"
	"github.com/skrashevich/telegram-mock-ai/internal/bot"
	"github.com/skrashevich/telegram-mock-ai/internal/models"
)

// handleGetFile returns a mock File object for any file_id.
func (s *Server) handleGetFile(w http.ResponseWriter, r *http.Request, b *bot.Bot) {
	fileID := parseStringParam(r, "file_id")
	if fileID == "" {
		respondError(w, http.StatusBadRequest, "Bad Request: file_id is required")
		return
	}

	fileUniqueID := fileID
	if len(fileUniqueID) > 20 {
		fileUniqueID = fileUniqueID[:20]
	}

	filePath := guessFilePath(fileID)

	respondOK(w, models.File{
		FileID:       fileID,
		FileUniqueID: fileUniqueID,
		FileSize:     0,
		FilePath:     filePath,
	})
}

// handleFileDownload serves generated placeholder files at /file/bot{token}/{file_path}.
func (s *Server) handleFileDownload(w http.ResponseWriter, r *http.Request) {
	// Path: /file/bot{token}/{dir}/{filename}
	path := r.URL.Path
	if !strings.HasPrefix(path, "/file/bot") {
		http.NotFound(w, r)
		return
	}

	rest := path[len("/file/bot"):]
	slashIdx := strings.Index(rest, "/")
	if slashIdx < 0 {
		http.NotFound(w, r)
		return
	}

	filePath := rest[slashIdx+1:] // e.g. "photos/file_abc123.jpg"
	slog.Debug("file download", "file_path", filePath)

	switch {
	case strings.HasPrefix(filePath, "photos/"):
		servePlaceholderJPEG(w, filePath, 800, 600)
	case strings.HasPrefix(filePath, "stickers/"):
		servePlaceholderWebP(w, filePath, 512, 512)
	case strings.HasPrefix(filePath, "videos/"):
		// Return a thumbnail-like JPEG for video
		servePlaceholderJPEG(w, filePath, 640, 480)
	case strings.HasPrefix(filePath, "documents/"):
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename=\"document.pdf\"")
		w.Write([]byte("%PDF-1.4 mock document"))
	case strings.HasPrefix(filePath, "voice/"):
		w.Header().Set("Content-Type", "audio/ogg")
		w.Write([]byte("OggS mock voice"))
	case strings.HasPrefix(filePath, "music/"):
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write([]byte("ID3 mock audio"))
	default:
		http.NotFound(w, r)
	}
}

// servePlaceholderJPEG generates a colorful placeholder JPEG with a pattern.
func servePlaceholderJPEG(w http.ResponseWriter, seed string, width, height int) {
	img := generatePlaceholderImage(seed, width, height)
	w.Header().Set("Content-Type", "image/jpeg")
	jpeg.Encode(w, img, &jpeg.Options{Quality: 85})
}

// servePlaceholderPNG generates a placeholder PNG.
func servePlaceholderPNG(w http.ResponseWriter, seed string, width, height int) {
	img := generatePlaceholderImage(seed, width, height)
	w.Header().Set("Content-Type", "image/png")
	var buf bytes.Buffer
	png.Encode(&buf, img)
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.Write(buf.Bytes())
}

// servePlaceholderWebP generates a placeholder WebP with transparency (for stickers).
func servePlaceholderWebP(w http.ResponseWriter, seed string, width, height int) {
	img := generateStickerImage(seed, width, height)
	w.Header().Set("Content-Type", "image/webp")
	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, &webp.Options{Lossy: false}); err != nil {
		slog.Error("webp encode failed, falling back to PNG", "err", err)
		w.Header().Set("Content-Type", "image/png")
		buf.Reset()
		png.Encode(&buf, img)
	}
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.Write(buf.Bytes())
}

// generatePlaceholderImage creates a colorful gradient image seeded by the file path.
func generatePlaceholderImage(seed string, width, height int) *image.RGBA {
	h := sha256.Sum256([]byte(seed))

	// Pick two colors from the hash for a gradient
	c1 := color.RGBA{R: h[0], G: h[1], B: h[2], A: 255}
	c2 := color.RGBA{R: h[3], G: h[4], B: h[5], A: 255}

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		t := float64(y) / float64(height)
		for x := 0; x < width; x++ {
			// Add a subtle wave pattern
			wave := math.Sin(float64(x)/30.0+float64(h[6])) * 0.1
			t2 := math.Min(1, math.Max(0, t+wave))
			r := uint8(float64(c1.R)*(1-t2) + float64(c2.R)*t2)
			g := uint8(float64(c1.G)*(1-t2) + float64(c2.G)*t2)
			b := uint8(float64(c1.B)*(1-t2) + float64(c2.B)*t2)
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}

	// Draw a simple shape in the center
	centerX, centerY := width/2, height/2
	radius := min(width, height) / 4
	shapeColor := color.RGBA{R: 255 - h[7], G: 255 - h[8], B: 255 - h[9], A: 180}

	for y := centerY - radius; y <= centerY+radius; y++ {
		for x := centerX - radius; x <= centerX+radius; x++ {
			dx := float64(x - centerX)
			dy := float64(y - centerY)
			if dx*dx+dy*dy <= float64(radius*radius) {
				blendPixel(img, x, y, shapeColor)
			}
		}
	}

	return img
}

// generateStickerImage creates a simple shape on a transparent background.
func generateStickerImage(seed string, width, height int) *image.NRGBA {
	h := sha256.Sum256([]byte(seed))
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	// Transparent background (already zero-initialized = transparent)

	// Draw a filled rounded shape
	centerX, centerY := width/2, height/2
	radius := min(width, height) * 2 / 5

	mainColor := color.NRGBA{R: h[0], G: h[1], B: h[2], A: 255}
	outlineColor := color.NRGBA{R: h[0] / 2, G: h[1] / 2, B: h[2] / 2, A: 255}

	// Fill circle
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dx := float64(x - centerX)
			dy := float64(y - centerY)
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist <= float64(radius-3) {
				img.Set(x, y, mainColor)
			} else if dist <= float64(radius) {
				img.Set(x, y, outlineColor)
			}
		}
	}

	// Draw simple eyes and mouth for an emoji-like sticker
	eyeColor := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	pupilColor := color.NRGBA{R: 30, G: 30, B: 30, A: 255}

	eyeR := radius / 8
	pupilR := eyeR / 2

	// Left eye
	drawCircle(img, centerX-radius/3, centerY-radius/4, eyeR, eyeColor)
	drawCircle(img, centerX-radius/3, centerY-radius/4, pupilR, pupilColor)
	// Right eye
	drawCircle(img, centerX+radius/3, centerY-radius/4, eyeR, eyeColor)
	drawCircle(img, centerX+radius/3, centerY-radius/4, pupilR, pupilColor)

	// Mouth - simple arc
	for angle := 0.3; angle < math.Pi-0.3; angle += 0.02 {
		mx := centerX + int(float64(radius/3)*math.Cos(angle))
		my := centerY + radius/4 + int(float64(radius/5)*math.Sin(angle))
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				img.Set(mx+dx, my+dy, pupilColor)
			}
		}
	}

	return img
}

func drawCircle(img draw.Image, cx, cy, r int, c color.Color) {
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx := float64(x - cx)
			dy := float64(y - cy)
			if dx*dx+dy*dy <= float64(r*r) {
				img.Set(x, y, c)
			}
		}
	}
}

func blendPixel(img *image.RGBA, x, y int, c color.RGBA) {
	if x < 0 || x >= img.Bounds().Dx() || y < 0 || y >= img.Bounds().Dy() {
		return
	}
	existing := img.RGBAAt(x, y)
	alpha := float64(c.A) / 255.0
	img.Set(x, y, color.RGBA{
		R: uint8(float64(existing.R)*(1-alpha) + float64(c.R)*alpha),
		G: uint8(float64(existing.G)*(1-alpha) + float64(c.G)*alpha),
		B: uint8(float64(existing.B)*(1-alpha) + float64(c.B)*alpha),
		A: 255,
	})
}

// guessFilePath returns a plausible file_path based on Telegram file_id prefixes.
func guessFilePath(fileID string) string {
	hex := models.RandomHex(8)
	switch {
	case strings.HasPrefix(fileID, "AgAC"): // photos
		return "photos/file_" + hex + ".jpg"
	case strings.HasPrefix(fileID, "CAAC"): // stickers
		return "stickers/file_" + hex + ".webp"
	case strings.HasPrefix(fileID, "BAADAgAD"): // videos
		return "videos/file_" + hex + ".mp4"
	case strings.HasPrefix(fileID, "BQAC"): // documents
		return "documents/file_" + hex + ".pdf"
	case strings.HasPrefix(fileID, "CQACAgIAAxkBAAI"): // audio
		return "music/file_" + hex + ".mp3"
	case strings.HasPrefix(fileID, "DQAC"): // voice
		return "voice/file_" + hex + ".ogg"
	default:
		return "documents/file_" + hex
	}
}

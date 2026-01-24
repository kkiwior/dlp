package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"video-microservice/internal/streamer"
	"video-microservice/internal/ytdlp"
)

func main() {
	http.HandleFunc("/video", videoHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func videoHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	query := r.URL.Query()
	url := query.Get("url")
	if url == "" {
		http.Error(w, "Missing 'url' parameter", http.StatusBadRequest)
		return
	}

	qParam := query.Get("quality")
	var quality ytdlp.Quality
	switch qParam {
	case "low":
		quality = ytdlp.QualityLow
	case "medium":
		quality = ytdlp.QualityMedium
	case "high":
		quality = ytdlp.QualityHigh
	default:
		// Default to high if not specified or invalid
		quality = ytdlp.QualityHigh
	}

	log.Printf("Processing request for URL: %s, Quality: %s", url, quality)

	// Get Video Info
	info, err := ytdlp.GetVideoInfo(ctx, url)
	if err != nil {
		if errors.Is(err, ytdlp.ErrVideoNotFound) {
			http.Error(w, "Video not found", http.StatusNotFound)
			return
		}
		log.Printf("Error getting video info: %v", err)
		http.Error(w, "Failed to fetch video metadata", http.StatusInternalServerError)
		return
	}

	// Select Formats
	video, audio := ytdlp.SelectFormats(info, quality)
	if video == nil {
		http.Error(w, "No suitable video format found", http.StatusNotFound)
		return
	}

	// Log selection
	audioUrl := ""
	audioCodec := ""
	if audio != nil {
		audioUrl = audio.URL
		audioCodec = audio.ACodec
		log.Printf("Selected Video: %s (%dp, %s), Audio: %s (%s)",
			video.FormatID, video.Height, video.VCodec, audio.FormatID, audio.ACodec)
	} else {
		log.Printf("Selected Video: %s (%dp, %s), No separate audio",
			video.FormatID, video.Height, video.VCodec)
	}

	// Set Headers
	w.Header().Set("Content-Type", "video/mp4")
	// Disable buffering in some proxies/clients?
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// Stream
	// Note: If audio is nil, audioUrl is empty string, handling inside streamer
	err = streamer.StreamVideo(ctx, video.URL, audioUrl, video.VCodec, audioCodec, w)
	if err != nil {
		// If we already wrote headers (likely), this error will just log to server console
		// and client will see a truncated stream.
		log.Printf("Streaming error: %v", err)
		return
	}

	log.Println("Streaming completed successfully")
}

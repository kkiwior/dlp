package streamer

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

// StreamVideo starts the ffmpeg process to stream the content
func StreamVideo(ctx context.Context, videoURL string, videoHeaders map[string]string, audioURL string, audioHeaders map[string]string, vCodec, aCodec string, w io.Writer) error {
	args := []string{
		"-hide_banner",
		"-loglevel", "error",
	}

	// Add inputs
	// Input 0: Video
	args = append(args, argsFromHeaders(videoHeaders)...)
	args = append(args, "-i", videoURL)

	hasSeparateAudio := audioURL != "" && audioURL != videoURL
	if hasSeparateAudio {
		// Input 1: Audio
		args = append(args, argsFromHeaders(audioHeaders)...)
		args = append(args, "-i", audioURL)
	}

	// Map streams
	if hasSeparateAudio {
		args = append(args, "-map", "0:v:0", "-map", "1:a:0")
	} else {
		// Single input with both (or just video)
		args = append(args, "-map", "0:v:0")
		// Check if the single input has audio? We assume yes if passed.
		// However, if we just want to be safe, we map audio if available.
		// But explicit map is better.
		args = append(args, "-map", "0:a:0?") // ? means optional
	}

	// Video Codec settings
	// User requirement: "output encoded in h264".
	// If source is already h264 (avc1), we copy.
	if strings.Contains(strings.ToLower(vCodec), "avc1") || strings.Contains(strings.ToLower(vCodec), "h264") {
		args = append(args, "-c:v", "copy")
	} else {
		// Transcode to H264
		// -preset superfast to be efficient but decent size.
		// -tune zerolatency for streaming.
		args = append(args, "-c:v", "libx264", "-preset", "superfast", "-tune", "zerolatency")
	}

	// Audio Codec settings
	if strings.Contains(strings.ToLower(aCodec), "mp4a") || strings.Contains(strings.ToLower(aCodec), "aac") {
		args = append(args, "-c:a", "copy")
	} else {
		args = append(args, "-c:a", "aac")
	}

	// Output format settings for streaming MP4
	args = append(args, "-f", "mp4", "-movflags", "frag_keyframe+empty_moov", "pipe:1")

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	// Bind stdout to the writer (HTTP response)
	cmd.Stdout = w

	// Bind stderr to OS stderr so we can see logs in container
	cmd.Stderr = os.Stderr

	log.Printf("Starting ffmpeg with args: %v", args)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg execution failed: %w", err)
	}

	return nil
}

func argsFromHeaders(headers map[string]string) []string {
	var args []string
	var headerList []string
	for k, v := range headers {
		if strings.EqualFold(k, "User-Agent") {
			args = append(args, "-user_agent", v)
		} else {
			headerList = append(headerList, fmt.Sprintf("%s: %s", k, v))
		}
	}
	if len(headerList) > 0 {
		// CRLF separated
		headerStr := strings.Join(headerList, "\r\n") + "\r\n"
		args = append(args, "-headers", headerStr)
	}
	return args
}

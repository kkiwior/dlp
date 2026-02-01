package streamer

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

type monitoringWriter struct {
	w     io.Writer
	start time.Time
	first bool
}

func (mw *monitoringWriter) Write(p []byte) (n int, err error) {
	if !mw.first {
		mw.first = true
		log.Printf("Streamer: First byte sent to client after %v", time.Since(mw.start))
	}
	return mw.w.Write(p)
}

// StreamVideo starts the ffmpeg process to stream the content
func StreamVideo(ctx context.Context, videoURL string, videoHeaders map[string]string, audioURL string, audioHeaders map[string]string, vCodec, aCodec string, w io.Writer) error {
	args := buildFfmpegArgs(videoURL, videoHeaders, audioURL, audioHeaders, vCodec, aCodec)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	// Wrap writer to monitor TTFB
	mw := &monitoringWriter{w: w, start: time.Now()}
	cmd.Stdout = mw

	// Pipe stderr to capture progress
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to pipe stderr: %w", err)
	}

	log.Printf("Starting ffmpeg with args: %v", args)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ffmpeg start failed: %w", err)
	}

	// Read stderr in a goroutine
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderrPipe.Read(buf)
			if n > 0 {
				chunk := buf[:n]
				os.Stderr.Write(chunk) // Pass through to original stderr

				// Simple heuristic: if we see "speed=", log it as a distinct log line for visibility
				s := string(chunk)
				if strings.Contains(s, "speed=") {
					// Extract the line or just log the chunk.
					// Since chunk might be partial, this isn't perfect, but good enough for debug.
					// We'll log it if it looks like a stats line.
					log.Printf("FFMPEG PROGRESS: %s", strings.TrimSpace(s))
				}
			}
			if err != nil {
				break
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ffmpeg execution failed: %w", err)
	}

	return nil
}

func buildFfmpegArgs(videoURL string, videoHeaders map[string]string, audioURL string, audioHeaders map[string]string, vCodec, aCodec string) []string {
	args := []string{
		"-hide_banner",
		"-loglevel", "info",
		"-threads", "0",
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
	// If source is already h264 (avc1) or h265 (hevc), we copy.
	vCodecLower := strings.ToLower(vCodec)
	if strings.Contains(vCodecLower, "avc1") || strings.Contains(vCodecLower, "h264") ||
		strings.Contains(vCodecLower, "hevc") || strings.Contains(vCodecLower, "hvc1") || strings.Contains(vCodecLower, "hev1") || strings.Contains(vCodecLower, "h265") {
		args = append(args, "-c:v", "copy")
	} else {
		// Transcode to H264
		// -preset ultrafast to be efficient but decent size.
		// We remove zerolatency to allow better buffering/throughput.
		// We add -g 60 to force keyframes every ~2s (assuming 30fps) for frequent fragmentation.
		// -sc_threshold 0 ensures strict GOP adherence.
		args = append(args, "-c:v", "libx264", "-preset", "ultrafast", "-g", "60", "-keyint_min", "60", "-sc_threshold", "0")
	}

	// Audio Codec settings
	if strings.Contains(strings.ToLower(aCodec), "mp4a") || strings.Contains(strings.ToLower(aCodec), "aac") {
		args = append(args, "-c:a", "copy")
	} else {
		args = append(args, "-c:a", "aac")
	}

	// Output format settings for streaming MP4
	args = append(args, "-f", "mp4", "-movflags", "frag_keyframe+empty_moov", "pipe:1")

	return args
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

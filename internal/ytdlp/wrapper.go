package ytdlp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"time"
)

var (
	infoCache sync.Map
)

type cachedInfo struct {
	info      *Info
	timestamp time.Time
}

const cacheTTL = 10 * time.Minute

func init() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			infoCache.Range(func(key, value interface{}) bool {
				entry, ok := value.(cachedInfo)
				if !ok {
					infoCache.Delete(key)
					return true
				}
				if time.Since(entry.timestamp) > cacheTTL {
					infoCache.Delete(key)
				}
				return true
			})
		}
	}()
}
var ErrVideoNotFound = errors.New("video not found")

// Format represents a single stream format
type Format struct {
	FormatID    string            `json:"format_id"`
	URL         string            `json:"url"`
	VCodec      string            `json:"vcodec"`
	ACodec      string            `json:"acodec"`
	Width       int               `json:"width,omitempty"`
	Height      int               `json:"height,omitempty"`
	TBR         float64           `json:"tbr,omitempty"` // Total bitrate
	ABR         float64           `json:"abr,omitempty"` // Audio bitrate
	Protocol    string            `json:"protocol,omitempty"`
	HTTPHeaders map[string]string `json:"http_headers"`
}

// Info represents the video metadata
type Info struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Formats     []Format          `json:"formats"`
	HTTPHeaders map[string]string `json:"http_headers"`
}

// Quality enum
type Quality string

const (
	QualityLow    Quality = "low"
	QualityMedium Quality = "medium"
	QualityHigh   Quality = "high"
)

// GetVideoInfo fetches metadata for the given URL
func GetVideoInfo(ctx context.Context, videoURL string) (*Info, error) {
	if val, ok := infoCache.Load(videoURL); ok {
		entry, ok := val.(cachedInfo)
		if ok && time.Since(entry.timestamp) < cacheTTL {
			log.Printf("Cache HIT for URL: %s", videoURL)
			return entry.info, nil
		}
		infoCache.Delete(videoURL)
	}
	log.Printf("Cache MISS for URL: %s", videoURL)

	cmd := exec.CommandContext(ctx, "yt-dlp", "-J", "--no-playlist", videoURL)
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "Video unavailable") || strings.Contains(stderr, "HTTP Error 404") {
				return nil, ErrVideoNotFound
			}
		}
		return nil, fmt.Errorf("failed to run yt-dlp: %w", err)
	}

	var info Info
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	infoCache.Store(videoURL, cachedInfo{info: &info, timestamp: time.Now()})

	return &info, nil
}

// SelectFormats chooses the best video and audio formats based on quality
func SelectFormats(info *Info, quality Quality) (video *Format, audio *Format) {
	// Filter video and audio formats
	videos := make([]Format, 0, len(info.Formats))
	audios := make([]Format, 0, len(info.Formats))

	for _, f := range info.Formats {
		isVideo := f.VCodec != "none" && f.Width > 0
		isAudio := f.ACodec != "none"

		// Some formats are container only or video-only or audio-only
		// We prefer separate streams usually for high quality, but mixed is fine too if it matches
		if isVideo {
			videos = append(videos, f)
		}
		if isAudio {
			audios = append(audios, f)
		}
	}

	// Sort videos by bitrate (quality) descending
	slices.SortFunc(videos, func(a, b Format) int {
		// If resolution is different, prefer higher resolution
		if a.Height != b.Height {
			return b.Height - a.Height
		}
		// If resolution is same, prefer H264 (avc1) to avoid transcoding
		aH264 := strings.HasPrefix(a.VCodec, "avc1")
		bH264 := strings.HasPrefix(b.VCodec, "avc1")
		if aH264 != bH264 {
			if aH264 {
				return -1
			}
			return 1
		}
		// Otherwise bitrate
		return int(b.TBR - a.TBR)
	})

	// Sort audios by quality
	slices.SortFunc(audios, func(a, b Format) int {
		// 1. Prefer Audio Only (VCodec == "none")
		aAudioOnly := a.VCodec == "none"
		bAudioOnly := b.VCodec == "none"
		if aAudioOnly != bAudioOnly {
			if aAudioOnly {
				return -1
			}
			return 1
		}

		// 2. Prefer HTTPS over m3u8 (HLS)
		// m3u8 streams often require complex header handling or cookie propagation for segments which can fail.
		aHttps := strings.HasPrefix(a.Protocol, "http") && !strings.Contains(a.Protocol, "m3u8")
		bHttps := strings.HasPrefix(b.Protocol, "http") && !strings.Contains(b.Protocol, "m3u8")
		if aHttps != bHttps {
			if aHttps {
				return -1
			}
			return 1
		}

		// 3. Prefer Higher ABR (or TBR if ABR missing)
		aRate := a.ABR
		if aRate == 0 {
			aRate = a.TBR
		}
		bRate := b.ABR
		if bRate == 0 {
			bRate = b.TBR
		}
		return int(bRate - aRate)
	})

	// Select Video
	if len(videos) > 0 {
		switch quality {
		case QualityHigh:
			video = &videos[0]
		case QualityMedium:
			// Aim for 720p or closest
			video = findClosestResolution(videos, 720)
		case QualityLow:
			// Aim for 360p or lowest
			video = findClosestResolution(videos, 360)
		default:
			video = &videos[0]
		}
	}

	// Select Audio
	// Just pick best audio usually, unless we want to save bandwidth on low quality
	if len(audios) > 0 {
		if quality == QualityLow {
             // Pick lowest bitrate audio
             audio = &audios[len(audios)-1]
		} else {
             audio = &audios[0]
		}
	} else {
		// Fallback: if video format contains audio (pre-merged), use it as audio source too
		// But in our pipeline we treat them as inputs.
		// If video struct has ACodec != none, it has audio.
		if video != nil && video.ACodec != "none" {
			audio = video
		}
	}

    // Refinement: If we picked a video that is NOT H264, check if there is an H264 option
    // with the SAME height and similar bitrate (or just exists).
    // The sort logic above already puts H264 first if heights are equal.
    // So video[0] for a given height bucket is already the H264 one if available.
    // e.g. if we have [1080p VP9, 1080p H264], sorting by height (equal) -> H264 (prio) -> H264 wins.
    // Wait, my sort logic:
    // if height != -> height desc.
    // if height == -> H264 prio.
    // So yes, we already prioritize H264 for the SAME resolution.
    // But what if High Quality (Max) finds 4K VP9 (Height 2160) and 1080p H264 (Height 1080).
    // The sort puts 4K first. We pick 4K. We will transcode. This is correct behavior for "Max Quality".

	return video, audio
}

func findClosestResolution(videos []Format, targetHeight int) *Format {
	best := &videos[0]
	minDiff := abs(best.Height - targetHeight)

	for i := range videos {
		diff := abs(videos[i].Height - targetHeight)
		if diff < minDiff {
			minDiff = diff
			best = &videos[i]
		} else if diff == minDiff {
			// If equal diff (e.g. same resolution candidates), the list is already sorted by H264 preference then bitrate
			// So we stick with the earlier one (which is better due to sort)
		}
	}
	return best
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

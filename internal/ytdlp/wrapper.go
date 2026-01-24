package ytdlp

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// Format represents a single stream format
type Format struct {
	FormatID string  `json:"format_id"`
	URL      string  `json:"url"`
	VCodec   string  `json:"vcodec"`
	ACodec   string  `json:"acodec"`
	Width    int     `json:"width,omitempty"`
	Height   int     `json:"height,omitempty"`
	TBR      float64 `json:"tbr,omitempty"` // Total bitrate
}

// Info represents the video metadata
type Info struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Formats []Format `json:"formats"`
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
	cmd := exec.CommandContext(ctx, "yt-dlp", "-J", videoURL)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run yt-dlp: %w", err)
	}

	var info Info
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return &info, nil
}

// SelectFormats chooses the best video and audio formats based on quality
func SelectFormats(info *Info, quality Quality) (video *Format, audio *Format) {
	// Filter video and audio formats
	var videos []Format
	var audios []Format

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
	sort.Slice(videos, func(i, j int) bool {
		// If resolution is different, prefer higher resolution
		if videos[i].Height != videos[j].Height {
			return videos[i].Height > videos[j].Height
		}
		// If resolution is same, prefer H264 (avc1) to avoid transcoding
		iH264 := strings.HasPrefix(videos[i].VCodec, "avc1")
		jH264 := strings.HasPrefix(videos[j].VCodec, "avc1")
		if iH264 && !jH264 {
			return true
		}
		if !iH264 && jH264 {
			return false
		}
		// Otherwise bitrate
		return videos[i].TBR > videos[j].TBR
	})

	// Sort audios by bitrate descending
	sort.Slice(audios, func(i, j int) bool {
		return audios[i].TBR > audios[j].TBR
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

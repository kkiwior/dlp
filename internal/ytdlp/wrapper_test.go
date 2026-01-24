package ytdlp

import (
	"testing"
)

func TestSelectFormats(t *testing.T) {
	formats := []Format{
		{FormatID: "1", VCodec: "vp9", ACodec: "none", Width: 3840, Height: 2160, TBR: 5000},         // 4K VP9
		{FormatID: "2", VCodec: "avc1.640028", ACodec: "none", Width: 1920, Height: 1080, TBR: 3000}, // 1080p H264
		{FormatID: "3", VCodec: "vp9", ACodec: "none", Width: 1920, Height: 1080, TBR: 2800},         // 1080p VP9
		{FormatID: "4", VCodec: "avc1.4d401e", ACodec: "none", Width: 1280, Height: 720, TBR: 1500},  // 720p H264
		{FormatID: "5", VCodec: "vp9", ACodec: "none", Width: 640, Height: 360, TBR: 800},            // 360p VP9
		{FormatID: "audio1", VCodec: "none", ACodec: "mp4a.40.2", TBR: 128},
		{FormatID: "audio2", VCodec: "none", ACodec: "opus", TBR: 160},
	}

	info := &Info{
		Formats: formats,
	}

	// Test 1: High Quality -> Should pick 4K VP9 (Highest Res)
	v, a := SelectFormats(info, QualityHigh)
	if v.FormatID != "1" {
		t.Errorf("High Quality: Expected video 1 (4K VP9), got %s", v.FormatID)
	}
	if a.FormatID != "audio2" { // Audio sorted by bitrate desc -> opus 160
		t.Errorf("High Quality: Expected audio2, got %s", a.FormatID)
	}

	// Test 2: Medium Quality -> Should pick ~720p. format 4 is 720p H264.
	v, a = SelectFormats(info, QualityMedium)
	if v.FormatID != "4" {
		t.Errorf("Medium Quality: Expected video 4 (720p H264), got %s", v.FormatID)
	}

	// Test 3: Low Quality -> Should pick ~360p. format 5 is 360p VP9.
	v, a = SelectFormats(info, QualityLow)
	if v.FormatID != "5" {
		t.Errorf("Low Quality: Expected video 5 (360p VP9), got %s", v.FormatID)
	}

	// Test 4: H264 Preference at same resolution
	// Case: 1080p H264 (2) vs 1080p VP9 (3).
	// If we ask for something that targets 1080p, or if we remove 4K option.

	formats2 := []Format{
		{FormatID: "2", VCodec: "avc1.640028", ACodec: "none", Width: 1920, Height: 1080, TBR: 3000}, // 1080p H264
		{FormatID: "3", VCodec: "vp9", ACodec: "none", Width: 1920, Height: 1080, TBR: 3500},         // 1080p VP9 (Higher bitrate!)
		// Even if VP9 has higher bitrate, our logic prioritizes H264 for same resolution
	}
	info2 := &Info{Formats: formats2}

	v, _ = SelectFormats(info2, QualityHigh)
	if v.FormatID != "2" {
		t.Errorf("H264 Preference: Expected video 2 (1080p H264), got %s", v.FormatID)
	}
}

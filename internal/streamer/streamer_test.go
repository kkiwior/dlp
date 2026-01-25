package streamer

import (
	"strings"
	"testing"
)

func TestBuildFfmpegArgs_Transcode(t *testing.T) {
	// Arrange
	videoURL := "http://example.com/video"
	videoHeaders := map[string]string{"User-Agent": "TestAgent"}
	audioURL := "http://example.com/audio"
	audioHeaders := map[string]string{"User-Agent": "TestAgent"}
	vCodec := "vp9" // Requires transcoding
	aCodec := "aac"

	// Act
	args := buildFfmpegArgs(videoURL, videoHeaders, audioURL, audioHeaders, vCodec, aCodec)

	// Assert
	argsStr := strings.Join(args, " ")

	// Check for transcoding flags
	if !strings.Contains(argsStr, "-c:v libx264") {
		t.Errorf("Expected transcoding to libx264, got args: %v", args)
	}

	// Check for preset (expected behavior is ultrafast)
	if !strings.Contains(argsStr, "-preset ultrafast") {
		t.Errorf("Expected preset ultrafast, got args: %v", args)
	}

    // Check for threads (expected behavior is -threads 0)
    if !strings.Contains(argsStr, "-threads 0") {
        t.Errorf("Expected -threads 0 flag, got args: %v", args)
    }
}

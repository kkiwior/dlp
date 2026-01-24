//go:build integration

package ytdlp

import (
	"context"
	"testing"
)

func TestGetVideoInfo_NotFound(t *testing.T) {
	// This test relies on yt-dlp being installed and available in the path.
	// It attempts to fetch info for a non-existent URL.

	ctx := context.Background()
	// Using a generic URL that is guaranteed to be 404 or a non-existent YouTube video ID
	url := "https://www.youtube.com/watch?v=zzzzzzzzzzz"

	info, err := GetVideoInfo(ctx, url)
	if info != nil {
		t.Errorf("Expected info to be nil, got %v", info)
	}
	if err != ErrVideoNotFound {
		t.Errorf("Expected ErrVideoNotFound, got %v", err)
	}

	// Also test generic 404
	url2 := "https://example.com/nothing"
	info2, err2 := GetVideoInfo(ctx, url2)
	if info2 != nil {
		t.Errorf("Expected info2 to be nil, got %v", info2)
	}
	if err2 != ErrVideoNotFound {
		t.Errorf("Expected ErrVideoNotFound, got %v", err2)
	}
}

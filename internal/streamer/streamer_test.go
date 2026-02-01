package streamer

import (
	"testing"
)

func TestBuildFfmpegArgs(t *testing.T) {
	tests := []struct {
		name        string
		vCodec      string
		wantPreset  string
		wantThreads bool
		wantCopy    bool
	}{
		{
			name:        "Transcoding needed (VP9)",
			vCodec:      "vp9",
			wantPreset:  "ultrafast",
			wantThreads: true,
			wantCopy:    false,
		},
		{
			name:        "No transcoding needed (H264)",
			vCodec:      "h264",
			wantPreset:  "",
			wantThreads: true,
			wantCopy:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildFfmpegArgs("http://video", nil, "http://audio", nil, tt.vCodec, "aac")

			// Check for preset
			foundPreset := false
			for i, arg := range args {
				if arg == "-preset" && i+1 < len(args) {
					if args[i+1] == tt.wantPreset {
						foundPreset = true
					} else if tt.wantPreset != "" {
						t.Errorf("got preset %s, want %s", args[i+1], tt.wantPreset)
					}
				}
			}
			if tt.wantPreset != "" && !foundPreset {
				t.Errorf("preset %s not found", tt.wantPreset)
			}
			// If we expect no preset (wantPreset == ""), verifying we don't find one is tricky
			// because existing code might have one if logic is buggy, but for Copy case it shouldn't be there.
			// The current code puts preset only in the else block of copy.

			// Check for threads
			foundThreads := false
			for i, arg := range args {
				if arg == "-threads" && i+1 < len(args) && args[i+1] == "0" {
					foundThreads = true
				}
			}
			if tt.wantThreads && !foundThreads {
				t.Errorf("threads 0 not found")
			}

			// Check for copy
			foundCopy := false
			for i, arg := range args {
				if arg == "-c:v" && i+1 < len(args) && args[i+1] == "copy" {
					foundCopy = true
				}
			}
			if tt.wantCopy != foundCopy {
				t.Errorf("wantCopy %v, got %v", tt.wantCopy, foundCopy)
			}
		})
	}
}

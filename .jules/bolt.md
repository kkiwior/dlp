## 2026-01-24 - Caching External Process Output
**Learning:** External processes like `yt-dlp` have significant startup and network latency (~3s). Caching their output is a high-impact optimization.
**Action:** Use `sync.Map` with a TTL for process output caching. Ensure background cleanup prevents memory leaks. Add flags like `--no-playlist` to prevent accidental massive fetches.

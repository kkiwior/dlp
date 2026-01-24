# Video Streaming Microservice

A high-performance Go microservice that streams videos from various platforms (supported by `yt-dlp`) directly to your client. It intelligently handles transcoding, preferring to pass-through H.264 streams when available, or real-time transcoding to H.264/MP4 otherwise.

## ⚠️ AI Disclosure

This code was written by **Jules** (an AI Agent), powered by the **Gemini 3 pro** model.

**Disclaimer:** This is "AI slop". It might work perfectly, or it might hallucinate dependencies, eat your RAM, or crash unexpectedly. Use at your own risk. Code quality is not guaranteed to meet human standards of sanity.

## Usage

### Endpoint

`GET /video`

### Parameters

| Parameter | Type   | Description                                                                 | Required |
| :-------- | :----- | :-------------------------------------------------------------------------- | :------- |
| `url`     | String | The URL of the video to stream (YouTube, Vimeo, etc.)                       | Yes      |
| `quality` | String | The desired quality. Options: `low`, `medium`, `high`. Defaults to `high`. | No       |

### Examples

**Stream a video in high quality:**
```bash
http://localhost:8080/video?url=https://www.youtube.com/watch?v=dQw4w9WgXcQ
```

**Stream in low quality (bandwidth saving):**
```bash
http://localhost:8080/video?url=https://www.youtube.com/watch?v=dQw4w9WgXcQ&quality=low
```

## Running with Docker

```bash
docker build -t video-microservice .
docker run -p 8080:8080 video-microservice
```

# Dry-Run Data Flow Documentation

> **Related Docs:**
> - [Integration Guide](./integration-guide.md) - Full system integration guide
> - [Project Execution Flow](./project-execution-flow.md) - Project execution and progress tracking

## Overview

This document describes the complete data flow for the dry-run keywords feature, showing how data transforms as it moves through each service:

1. **Crawler Worker** → **Collector**
2. **Collector** → **Project** (via webhook)
3. **Project** → **WebSocket** (via Redis Pub/Sub)

---

## Complete Flow Diagram

```
Client
  ↓ POST /projects/dryrun {"keywords": ["k1", "k2"]}
Project Service
  ↓ RabbitMQ: collector.inbound / crawler.dryrun_keyword
Collector Service
  ↓ Dispatch to YouTube/TikTok workers
Crawler Workers (YouTube/TikTok)
  ↓ Crawl 3 posts/keyword, 5 comments/post
  ↓ Return results
Collector Service (Results Handler)
  ↓ POST /internal/collector/dryrun/callback
Project Service (Webhook Handler)
  ↓ Redis Pub/Sub: user_noti:{user_id}
WebSocket Service
  ↓ WebSocket message
Client Browser
```

---

## Integration Guide

### Prerequisites

- Project Service running on port 8080
- Collector Service running on port 8081
- Redis running on port 6379
- RabbitMQ running on port 5672
- WebSocket Service running on port 8082

### Step-by-Step Integration Test

#### Step 1: Client → Project Service

```bash
# Request
curl -X POST http://localhost:8080/projects/dryrun \
  -H "Content-Type: application/json" \
  -H "Cookie: smap_auth_token=<jwt_token>" \
  -d '{
    "keywords": ["VinFast", "VF3"]
  }'

# Response
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "processing"
}
```

#### Step 2: Project Service → Redis (Job Mapping)

```bash
# Verify job mapping stored
redis-cli GET job:mapping:550e8400-e29b-41d4-a716-446655440000

# Expected
{"user_id":"user-uuid","project_id":"","created_at":"2025-12-05T10:00:00Z"}
```

#### Step 3: Project Service → RabbitMQ

```bash
# Message published to:
# Exchange: collector.inbound
# Routing Key: crawler.dryrun_keyword

# Message format:
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "task_type": "dryrun_keyword",
  "payload": {
    "keywords": ["VinFast", "VF3"],
    "limit_per_keyword": 3,
    "include_comments": true,
    "max_comments": 5
  },
  "attempt": 1,
  "max_attempts": 3,
  "emitted_at": "2025-12-05T10:00:00Z"
}
```

#### Step 4: Collector Service → Project Service (Webhook)

```bash
# Collector calls webhook after crawling
curl -X POST http://localhost:8080/internal/collector/dryrun/callback \
  -H "Content-Type: application/json" \
  -H "X-Internal-Key: <internal_key>" \
  -d '{
    "job_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "success",
    "platform": "tiktok",
    "payload": {
      "content": [...],
      "errors": []
    }
  }'

# Response
{"status": "ok"}
```

#### Step 5: Project Service → Redis Pub/Sub

```bash
# Subscribe to channel (for testing)
redis-cli SUBSCRIBE user_noti:user-uuid

# Message published:
{
  "type": "dryrun_result",
  "payload": {
    "job_id": "550e8400-e29b-41d4-a716-446655440000",
    "platform": "tiktok",
    "status": "success",
    "content": [...],
    "errors": []
  }
}
```

#### Step 6: WebSocket Service → Client

```javascript
// Client WebSocket connection
const ws = new WebSocket("ws://localhost:8082/ws?token=<jwt_token>");

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  if (message.type === "dryrun_result") {
    console.log("Dry-run result:", message.payload);
  }
};
```

### Error Scenarios

#### Scenario 1: Invalid Keywords

```bash
curl -X POST http://localhost:8080/projects/dryrun \
  -H "Content-Type: application/json" \
  -H "Cookie: smap_auth_token=<jwt_token>" \
  -d '{"keywords": []}'

# Response: 400 Bad Request
{"code": 30002, "message": "Wrong body"}
```

#### Scenario 2: Unauthorized

```bash
curl -X POST http://localhost:8080/projects/dryrun \
  -H "Content-Type: application/json" \
  -d '{"keywords": ["test"]}'

# Response: 401 Unauthorized
{"code": 30005, "message": "Unauthorized"}
```

#### Scenario 3: Crawler Failure

```bash
# Webhook callback with error
curl -X POST http://localhost:8080/internal/collector/dryrun/callback \
  -H "Content-Type: application/json" \
  -H "X-Internal-Key: <internal_key>" \
  -d '{
    "job_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "failed",
    "platform": "tiktok",
    "payload": {
      "content": [],
      "errors": [{"code": "RATE_LIMITED", "message": "Too many requests"}]
    }
  }'
```

### Configuration

#### Project Service (.env)

```env
# Redis (DB 0 for job mapping and pub/sub)
REDIS_HOST=localhost:6379
REDIS_PASSWORD=password

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# Internal API Key (for webhook authentication)
INTERNAL_KEY=your-internal-key
```

#### Collector Service

```env
# Project Service webhook URL
PROJECT_WEBHOOK_URL=http://localhost:8080/internal/collector/dryrun/callback
PROJECT_INTERNAL_KEY=your-internal-key
```

---

## CRAWLER WORKER → COLLECTOR

### Input to Collector (from Crawler Worker via RabbitMQ)

**Source:** Crawler Worker (YouTube/TikTok)  
**Destination:** Collector Results Handler  
**Transport:** RabbitMQ (results queue)  
**Reference:** MESSAGE-STRUCTURE-SPECIFICATION.md

#### Message Structure:

```json
{
  "success": true,
  "payload": [
    {
      "meta": {
        "id": "7234567890123456789",
        "platform": "tiktok",
        "job_id": "550e8400-e29b-41d4-a716-446655440000",
        "crawled_at": "2024-01-15T10:30:00Z",
        "published_at": "2024-01-10T08:00:00Z",
        "permalink": "https://www.tiktok.com/@user/video/7234567890123456789",
        "keyword_source": "cooking tutorial",
        "lang": "vi",
        "region": "VN",
        "pipeline_version": "crawler_tiktok_v3",
        "fetch_status": "success",
        "fetch_error": null
      },
      "content": {
        "text": "Easy cooking tutorial! #cooking #food",
        "duration": 45,
        "hashtags": ["cooking", "food"],
        "sound_name": "Original Sound - User",
        "category": "Food",
        "title": null,
        "media": {
          "type": "audio",
          "video_path": null,
          "audio_path": "tiktok/job-abc-123/7234567890123456789.mp3",
          "downloaded_at": "2024-01-15T10:31:00Z"
        },
        "transcription": "Today I will show you how to cook..."
      },
      "interaction": {
        "views": 150000,
        "likes": 12000,
        "comments_count": 450,
        "shares": 890,
        "saves": 2300,
        "engagement_rate": 0.0893,
        "updated_at": "2024-01-15T10:30:00Z"
      },
      "author": {
        "id": "user123",
        "name": "Cooking Master",
        "username": "cookingmaster",
        "followers": 500000,
        "following": 123,
        "likes": 5000000,
        "videos": 234,
        "is_verified": true,
        "bio": "Professional chef sharing recipes",
        "avatar_url": null,
        "profile_url": "https://www.tiktok.com/@cookingmaster",
        "country": null,
        "total_view_count": null
      },
      "comments": [
        {
          "id": "comment123",
          "parent_id": null,
          "post_id": "7234567890123456789",
          "user": {
            "id": null,
            "name": "FoodLover",
            "avatar_url": null
          },
          "text": "Amazing recipe!",
          "likes": 45,
          "replies_count": 2,
          "published_at": "2024-01-10T09:00:00Z",
          "is_author": false,
          "media": null,
          "is_favorited": false
        }
      ]
    }
  ]
}
```

#### Go Type Mapping (Collector Internal):

**File:** `collector/internal/results/uc_types.go`

```go
type CrawlerPayload struct {
    Success bool             `json:"success"`
    Payload []CrawlerContent `json:"payload"`
}

type CrawlerContent struct {
    Meta        CrawlerContentMeta        `json:"meta"`
    Content     CrawlerContentData        `json:"content"`
    Interaction CrawlerContentInteraction `json:"interaction"`
    Author      CrawlerContentAuthor      `json:"author"`
    Comments    []CrawlerComment          `json:"comments,omitempty"`
}

type CrawlerContentMeta struct {
    ID              string  `json:"id"`
    Platform        string  `json:"platform"`
    JobID           string  `json:"job_id"`
    CrawledAt       string  `json:"crawled_at"`   // RFC3339 string
    PublishedAt     string  `json:"published_at"` // RFC3339 string
    Permalink       string  `json:"permalink"`
    KeywordSource   string  `json:"keyword_source"`
    Lang            string  `json:"lang"`
    Region          string  `json:"region"`
    PipelineVersion string  `json:"pipeline_version"`
    FetchStatus     string  `json:"fetch_status"`
    FetchError      *string `json:"fetch_error,omitempty"`
}

type CrawlerContentData struct {
    Text          string               `json:"text"`
    Duration      int                  `json:"duration,omitempty"`
    Hashtags      []string             `json:"hashtags,omitempty"`
    SoundName     string               `json:"sound_name,omitempty"`
    Category      *string              `json:"category,omitempty"`
    Media         *CrawlerContentMedia `json:"media,omitempty"`
    Transcription *string              `json:"transcription,omitempty"`
}

type CrawlerContentMedia struct {
    Type         string `json:"type"`
    VideoPath    string `json:"video_path,omitempty"`
    AudioPath    string `json:"audio_path,omitempty"`
    DownloadedAt string `json:"downloaded_at,omitempty"` // RFC3339 string
}

type CrawlerContentInteraction struct {
    Views          int     `json:"views"`
    Likes          int     `json:"likes"`
    CommentsCount  int     `json:"comments_count"`
    Shares         int     `json:"shares"`
    Saves          int     `json:"saves,omitempty"`
    EngagementRate float64 `json:"engagement_rate,omitempty"`
    UpdatedAt      string  `json:"updated_at"` // RFC3339 string
}

type CrawlerContentAuthor struct {
    ID         string  `json:"id"`
    Name       string  `json:"name"`
    Username   string  `json:"username"`
    Followers  int     `json:"followers"`
    Following  int     `json:"following"`
    Likes      int     `json:"likes"`
    Videos     int     `json:"videos"`
    IsVerified bool    `json:"is_verified"`
    Bio        string  `json:"bio,omitempty"`
    AvatarURL  *string `json:"avatar_url,omitempty"`
    ProfileURL string  `json:"profile_url"`
}

type CrawlerComment struct {
    ID           string             `json:"id"`
    ParentID     *string            `json:"parent_id,omitempty"`
    PostID       string             `json:"post_id"`
    User         CrawlerCommentUser `json:"user"`
    Text         string             `json:"text"`
    Likes        int                `json:"likes"`
    RepliesCount int                `json:"replies_count"`
    PublishedAt  string             `json:"published_at"` // RFC3339 string
    IsAuthor     bool               `json:"is_author"`
    Media        *string            `json:"media,omitempty"`
}

type CrawlerCommentUser struct {
    ID        *string `json:"id,omitempty"`
    Name      string  `json:"name"`
    AvatarURL *string `json:"avatar_url,omitempty"`
}
```

### Processing in Collector

**File:** `collector/internal/results/usecase/result_uc.go`

**Steps:**

1. Receive `CrawlerResult` (contains `success` and `payload` as `any`)
2. Marshal `payload` to JSON
3. Unmarshal to `CrawlerPayload` struct
4. Extract `job_id` and `platform` from first item's meta
5. Validate required fields (job_id, platform, content_id)
6. Map each `CrawlerContent` to `project.Content`:
   - Parse RFC3339 timestamps → `time.Time`
   - Map all nested structures
   - Validate each field

**Key Transformations:**

- `string` timestamps → `time.Time` (RFC3339 parsing)
- All other fields remain the same
- Validation of required fields

---

## COLLECTOR → PROJECT

### Output from Collector (to Project via HTTP Webhook)

**Source:** Collector Results Handler  
**Destination:** Project Webhook Endpoint  
**Transport:** HTTP POST  
**Endpoint:** `POST /internal/collector/dryrun/callback`  
**Authentication:** `X-Internal-Key` header

#### Webhook Request Structure:

```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "success",
  "platform": "tiktok",
  "payload": {
    "content": [
      {
        "meta": {
          "id": "7234567890123456789",
          "platform": "tiktok",
          "job_id": "550e8400-e29b-41d4-a716-446655440000",
          "crawled_at": "2024-01-15T10:30:00Z",
          "published_at": "2024-01-10T08:00:00Z",
          "permalink": "https://www.tiktok.com/@user/video/7234567890123456789",
          "keyword_source": "cooking tutorial",
          "lang": "vi",
          "region": "VN",
          "pipeline_version": "crawler_tiktok_v3",
          "fetch_status": "success",
          "fetch_error": null
        },
        "content": {
          "text": "Easy cooking tutorial! #cooking #food",
          "duration": 45,
          "hashtags": ["cooking", "food"],
          "sound_name": "Original Sound - User",
          "category": "Food",
          "media": {
            "type": "audio",
            "video_path": "",
            "audio_path": "tiktok/job-abc-123/7234567890123456789.mp3",
            "downloaded_at": "2024-01-15T10:31:00Z"
          },
          "transcription": "Today I will show you how to cook..."
        },
        "interaction": {
          "views": 150000,
          "likes": 12000,
          "comments_count": 450,
          "shares": 890,
          "saves": 2300,
          "engagement_rate": 0.0893,
          "updated_at": "2024-01-15T10:30:00Z"
        },
        "author": {
          "id": "user123",
          "name": "Cooking Master",
          "username": "cookingmaster",
          "followers": 500000,
          "following": 123,
          "likes": 5000000,
          "videos": 234,
          "is_verified": true,
          "bio": "Professional chef sharing recipes",
          "avatar_url": null,
          "profile_url": "https://www.tiktok.com/@cookingmaster"
        },
        "comments": [
          {
            "id": "comment123",
            "parent_id": null,
            "post_id": "7234567890123456789",
            "user": {
              "id": null,
              "name": "FoodLover",
              "avatar_url": null
            },
            "text": "Amazing recipe!",
            "likes": 45,
            "replies_count": 2,
            "published_at": "2024-01-10T09:00:00Z",
            "is_author": false,
            "media": null
          }
        ]
      }
    ],
    "errors": []
  }
}
```

#### Go Type Mapping (Collector → Project):

**File:** `collector/pkg/project/types.go`

```go
type CallbackRequest struct {
    JobID    string          `json:"job_id"`
    Status   string          `json:"status"` // "success" or "failed"
    Platform string          `json:"platform"`
    Payload  CallbackPayload `json:"payload"`
}

type CallbackPayload struct {
    Content []Content `json:"content,omitempty"`
    Errors  []Error   `json:"errors,omitempty"`
}

type Content struct {
    Meta        ContentMeta        `json:"meta"`
    Content     ContentData        `json:"content"`
    Interaction ContentInteraction `json:"interaction"`
    Author      ContentAuthor      `json:"author"`
    Comments    []Comment          `json:"comments,omitempty"`
}

type ContentMeta struct {
    ID              string    `json:"id"`
    Platform        string    `json:"platform"`
    JobID           string    `json:"job_id"`
    CrawledAt       time.Time `json:"crawled_at"`      // time.Time (not string)
    PublishedAt     time.Time `json:"published_at"`    // time.Time (not string)
    Permalink       string    `json:"permalink"`
    KeywordSource   string    `json:"keyword_source"`
    Lang            string    `json:"lang"`
    Region          string    `json:"region"`
    PipelineVersion string    `json:"pipeline_version"`
    FetchStatus     string    `json:"fetch_status"`
    FetchError      *string   `json:"fetch_error"`
}

type ContentData struct {
    Text          string        `json:"text"`
    Duration      int           `json:"duration,omitempty"`
    Hashtags      []string      `json:"hashtags,omitempty"`
    SoundName     string        `json:"sound_name,omitempty"`
    Category      *string       `json:"category,omitempty"`
    Media         *ContentMedia `json:"media,omitempty"`
    Transcription *string       `json:"transcription,omitempty"`
}

type ContentMedia struct {
    Type         string    `json:"type"`
    VideoPath    string    `json:"video_path,omitempty"`
    AudioPath    string    `json:"audio_path,omitempty"`
    DownloadedAt time.Time `json:"downloaded_at,omitempty"` // time.Time (not string)
}

type ContentInteraction struct {
    Views          int       `json:"views"`
    Likes          int       `json:"likes"`
    CommentsCount  int       `json:"comments_count"`
    Shares         int       `json:"shares"`
    Saves          int       `json:"saves,omitempty"`
    EngagementRate float64   `json:"engagement_rate,omitempty"`
    UpdatedAt      time.Time `json:"updated_at"` // time.Time (not string)
}

type ContentAuthor struct {
    ID         string  `json:"id"`
    Name       string  `json:"name"`
    Username   string  `json:"username"`
    Followers  int     `json:"followers"`
    Following  int     `json:"following"`
    Likes      int     `json:"likes"`
    Videos     int     `json:"videos"`
    IsVerified bool    `json:"is_verified"`
    Bio        string  `json:"bio,omitempty"`
    AvatarURL  *string `json:"avatar_url,omitempty"`
    ProfileURL string  `json:"profile_url"`
}

type Comment struct {
    ID           string      `json:"id"`
    ParentID     *string     `json:"parent_id,omitempty"`
    PostID       string      `json:"post_id"`
    User         CommentUser `json:"user"`
    Text         string      `json:"text"`
    Likes        int         `json:"likes"`
    RepliesCount int         `json:"replies_count"`
    PublishedAt  time.Time   `json:"published_at"` // time.Time (not string)
    IsAuthor     bool        `json:"is_author"`
    Media        *string     `json:"media,omitempty"`
}

type CommentUser struct {
    ID        *string `json:"id,omitempty"`
    Name      string  `json:"name"`
    AvatarURL *string `json:"avatar_url,omitempty"`
}

type Error struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Keyword string `json:"keyword,omitempty"`
}
```

### Input to Project (Webhook Handler)

**File:** `project/internal/webhook/type.go`

**Note:** Project webhook types are **IDENTICAL** to collector's project types.

```go
// Same structure as collector/pkg/project/types.go
type CallbackRequest struct {
    JobID    string          `json:"job_id" binding:"required"`
    Status   string          `json:"status" binding:"required,oneof=success failed"`
    Platform string          `json:"platform" binding:"required,oneof=youtube tiktok"`
    Payload  CallbackPayload `json:"payload"`
    UserID   string          `json:"user_id,omitempty"` // Optional, looked up from Redis
}
// ... (rest identical to collector/pkg/project/types.go)
```

### Processing in Project

**File:** `project/internal/webhook/usecase/webhook.go`

**Steps:**

1. Receive `CallbackRequest` from collector
2. Lookup job mapping from Redis: `job_id` → `(user_id, project_id)`
3. Format Redis channel: `user_noti:{user_id}`
4. Construct message with type `"dryrun_result"`
5. Marshal to JSON
6. Publish to Redis Pub/Sub

**Key Transformations:**

- Lookup `user_id` from Redis job mapping
- Wrap in message envelope with type
- No data transformation (pass-through)

---

## PROJECT → WEBSOCKET

### Output from Project (to WebSocket via Redis Pub/Sub)

**Source:** Project Webhook Handler  
**Destination:** WebSocket Redis Subscriber  
**Transport:** Redis Pub/Sub  
**Channel:** `user_noti:{user_id}`

#### Redis Message Structure:

```json
{
  "type": "dryrun_result",
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "platform": "tiktok",
  "status": "success",
  "payload": {
    "content": [
      {
        "meta": {
          "id": "7234567890123456789",
          "platform": "tiktok",
          "job_id": "550e8400-e29b-41d4-a716-446655440000",
          "crawled_at": "2024-01-15T10:30:00Z",
          "published_at": "2024-01-10T08:00:00Z",
          "permalink": "https://www.tiktok.com/@user/video/7234567890123456789",
          "keyword_source": "cooking tutorial",
          "lang": "vi",
          "region": "VN",
          "pipeline_version": "crawler_tiktok_v3",
          "fetch_status": "success",
          "fetch_error": null
        },
        "content": {
          "text": "Easy cooking tutorial! #cooking #food",
          "duration": 45,
          "hashtags": ["cooking", "food"],
          "sound_name": "Original Sound - User",
          "category": "Food",
          "media": {
            "type": "audio",
            "video_path": "",
            "audio_path": "tiktok/job-abc-123/7234567890123456789.mp3",
            "downloaded_at": "2024-01-15T10:31:00Z"
          },
          "transcription": "Today I will show you how to cook..."
        },
        "interaction": {
          "views": 150000,
          "likes": 12000,
          "comments_count": 450,
          "shares": 890,
          "saves": 2300,
          "engagement_rate": 0.0893,
          "updated_at": "2024-01-15T10:30:00Z"
        },
        "author": {
          "id": "user123",
          "name": "Cooking Master",
          "username": "cookingmaster",
          "followers": 500000,
          "following": 123,
          "likes": 5000000,
          "videos": 234,
          "is_verified": true,
          "bio": "Professional chef sharing recipes",
          "avatar_url": null,
          "profile_url": "https://www.tiktok.com/@cookingmaster"
        },
        "comments": [
          {
            "id": "comment123",
            "parent_id": null,
            "post_id": "7234567890123456789",
            "user": {
              "id": null,
              "name": "FoodLover",
              "avatar_url": null
            },
            "text": "Amazing recipe!",
            "likes": 45,
            "replies_count": 2,
            "published_at": "2024-01-10T09:00:00Z",
            "is_author": false,
            "media": null
          }
        ]
      }
    ],
    "errors": []
  }
}
```

#### Go Type Mapping (Project → Redis):

**File:** `project/internal/webhook/usecase/webhook.go`

```go
// Constructed as map[string]interface{}
message := map[string]interface{}{
    "type":     "dryrun_result",
    "job_id":   req.JobID,
    "platform": req.Platform,
    "status":   req.Status,
    "payload":  req.Payload, // Pass-through from CallbackRequest
}
```

### Input to WebSocket (Redis Subscriber)

**File:** `websocket/internal/redis/subscriber.go`

#### Redis Message Type:

```go
type RedisMessage struct {
    Type    string          `json:"type"`
    Payload json.RawMessage `json:"payload"`
}
```

### Processing in WebSocket

**File:** `websocket/internal/redis/subscriber.go`

**Steps:**

1. Receive message from Redis channel `user_noti:*`
2. Extract `user_id` from channel name
3. Unmarshal to `RedisMessage`
4. Log dry-run specific info (job_id, platform, status)
5. Create WebSocket message
6. Send to Hub for delivery to user's connections

**Key Transformations:**

- Extract `user_id` from channel name
- Wrap in WebSocket message envelope
- Add timestamp
- No data transformation (pass-through)

### Output from WebSocket (to Client Browser)

**Source:** WebSocket Hub  
**Destination:** Client Browser  
**Transport:** WebSocket connection

#### WebSocket Message Structure:

```json
{
  "type": "dryrun_result",
  "payload": {
    "type": "dryrun_result",
    "job_id": "550e8400-e29b-41d4-a716-446655440000",
    "platform": "tiktok",
    "status": "success",
    "payload": {
      "content": [...],
      "errors": []
    }
  },
  "timestamp": "2024-01-15T10:30:05Z"
}
```

#### Go Type Mapping (WebSocket):

**File:** `websocket/internal/websocket/message.go`

```go
type MessageType string

const (
    MessageTypeDryRunResult MessageType = "dryrun_result"
)

type Message struct {
    Type      MessageType     `json:"type"`
    Payload   json.RawMessage `json:"payload"`
    Timestamp time.Time       `json:"timestamp"`
}
```

---

## Field-by-Field Comparison

### Key Differences

| Field Path                | Crawler → Collector      | Collector → Project | Project → WebSocket        |
| ------------------------- | ------------------------ | ------------------- | -------------------------- |
| **Timestamps**            | `string` (RFC3339)       | `time.Time`         | `string` (RFC3339 in JSON) |
| `meta.crawled_at`         | `"2024-01-15T10:30:00Z"` | `time.Time`         | `"2024-01-15T10:30:00Z"`   |
| `meta.published_at`       | `"2024-01-15T10:30:00Z"` | `time.Time`         | `"2024-01-15T10:30:00Z"`   |
| `interaction.updated_at`  | `"2024-01-15T10:30:00Z"` | `time.Time`         | `"2024-01-15T10:30:00Z"`   |
| `media.downloaded_at`     | `"2024-01-15T10:31:00Z"` | `time.Time`         | `"2024-01-15T10:31:00Z"`   |
| `comments[].published_at` | `"2024-01-10T09:00:00Z"` | `time.Time`         | `"2024-01-10T09:00:00Z"`   |

Note: Go's `time.Time` automatically marshals to RFC3339 string in JSON, so the wire format remains consistent.

### Fields That Remain Identical

All other fields remain identical across all services:

- All IDs (string)
- All counts (int)
- All text fields (string)
- All boolean fields (bool)
- All arrays and nested objects
- Platform-specific null handling

---

## Summary Table

| Service       | Input Format              | Internal Type                        | Output Format              | Key Transformation          |
| ------------- | ------------------------- | ------------------------------------ | -------------------------- | --------------------------- |
| **Collector** | JSON (string timestamps)  | `CrawlerContent` (string timestamps) | JSON (time.Time → RFC3339) | Parse RFC3339 → time.Time   |
| **Project**   | JSON (RFC3339 timestamps) | `webhook.Content` (time.Time)        | JSON (time.Time → RFC3339) | Pass-through (no transform) |
| **WebSocket** | JSON (RFC3339 timestamps) | `json.RawMessage`                    | JSON (pass-through)        | Envelope wrapping only      |

---

## Validation Checklist

- All field names match exactly across services
- All field types are compatible (string ↔ time.Time via JSON)
- NOT all required fields are present (see Missing Fields below)
- All optional fields use pointers correctly
- Platform-specific fields NOT fully handled (missing YouTube-only fields)
- Timestamp parsing/formatting is consistent (RFC3339)
- Error handling for missing/invalid fields
- Validation of required fields (job_id, platform, content_id)

---

## MISSING FIELDS - CRITICAL ISSUES

### Fields Missing from Implementation

The following fields from MESSAGE-STRUCTURE-SPECIFICATION.md are NOT implemented:

#### 1. ContentData - Missing `title` (YouTube only)

```go
// MISSING in all services
Title *string `json:"title,omitempty"`
```

Impact: YouTube video titles will be lost

#### 2. ContentAuthor - Missing YouTube-specific fields

```go
// MISSING in all services
Country        *string `json:"country,omitempty"`          // YouTube only
TotalViewCount *int    `json:"total_view_count,omitempty"` // YouTube only
```

Impact:

- YouTube channel country information will be lost
- YouTube channel total view count will be lost

#### 3. Comment - Missing `is_favorited` (YouTube only)

```go
// MISSING in all services
IsFavorited bool `json:"is_favorited"` // YouTube only
```

Impact: YouTube comment favorited status will be lost

### Files That Need Updates

1. **collector/internal/results/uc_types.go** - Add fields to internal types
2. **collector/pkg/project/types.go** - Add fields to webhook types
3. **project/internal/webhook/type.go** - Add fields to webhook types
4. **collector/internal/results/usecase/result_uc.go** - Update mapping logic

### Detailed Analysis

See **MISSING-FIELDS-ANALYSIS.md** for:

- Complete field-by-field comparison
- Required code changes
- Impact assessment
- Testing checklist

---

## Conclusion

Current Status: FUNCTIONALLY WORKING BUT DATA INCOMPLETE

The data flow is functionally operational but NOT 100% spec-compliant:

Working:

- Core flow (Client → Project → Collector → Workers → Collector → Project → WebSocket)
- Task dispatching and routing
- Webhook callbacks with retry
- Redis Pub/Sub fan-out
- WebSocket delivery
- Timestamp transformations (RFC3339 ↔ time.Time)
- TikTok-specific fields (sound_name, shares, saves, following, likes)
- Common fields (meta, content, interaction, author, comments)

Missing (YouTube-only fields):

- `content.title` - Video title
- `author.country` - Channel country
- `author.total_view_count` - Channel total views
- `comments[].is_favorited` - Comment favorited status

Recommendation: Add missing fields before production deployment to ensure 100% data integrity for YouTube content.

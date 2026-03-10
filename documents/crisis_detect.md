# Crisis Detection Configuration

## Phần 1: File Cấu hình JSON Mẫu

```json
{
  "project_id": "PRJ_VINAMILK_001",
  "config_version": "2.0",
  "last_updated": "2023-10-27T10:00:00Z",

  "triggers": {
    "keywords_trigger": {
      "enabled": true,
      "logic": "OR",
      "groups": [
        {
          "name": "critical_terms",
          "keywords": [
            "lừa đảo",
            "scam",
            "tẩy chay",
            "boycott",
            "fake",
            "giả mạo",
            "hàng nhái"
          ],
          "weight": 10
        },
        {
          "name": "legal_health_terms",
          "keywords": [
            "nhập viện",
            "ngộ độc",
            "kiện",
            "công an",
            "pháp luật",
            "cháy nổ"
          ],
          "weight": 15
        },
        {
          "name": "slang_terms",
          "keywords": ["phốt", "biến căng", "quay xe", "tráo hàng"],
          "weight": 5
        }
      ]
    },

    "volume_trigger": {
      "enabled": true,
      "metric": "mentions_count",
      "rules": [
        {
          "level": "WARNING",
          "threshold_percent_growth": 50,
          "comparison_window_hours": 4,
          "baseline": "average_last_7_days"
        },
        {
          "level": "CRITICAL",
          "threshold_percent_growth": 150,
          "comparison_window_hours": 1,
          "baseline": "average_last_24_hours"
        }
      ]
    },

    "sentiment_trigger": {
      "enabled": true,
      "min_sample_size": 50,
      "rules": [
        {
          "type": "negative_ratio",
          "threshold_percent": 30
        },
        {
          "type": "absa_aspect_alert",
          "critical_aspects": ["safety", "hygiene", "legal"],
          "negative_threshold_percent": 15
        }
      ]
    },

    "influencer_trigger": {
      "enabled": true,
      "logic": "OR",
      "rules": [
        {
          "type": "macro_influencer",
          "min_followers": 100000,
          "required_sentiment": "negative"
        },
        {
          "type": "viral_post",
          "min_shares": 500,
          "min_comments": 200,
          "required_sentiment": "negative"
        }
      ]
    }
  },

  "notification_channels": {
    "email": ["marketing_lead@brand.com", "pr_manager@brand.com"],
    "webhook_slack": "https://hooks.slack.com/services/...",
    "sms_alert": ["+84901234567"]
  }
}
```

---

## Phần 2: Giải thích Chi tiết & Logic Vận hành

Dưới đây là lý do tại sao các tham số trên lại cần thiết cho đồ án của bạn:

### 2. Keywords Trigger (Kích hoạt theo Từ khóa)

Thay vì gộp chung, chia làm 3 nhóm để hệ thống biết mức độ nghiêm trọng:

- **Critical Terms** — Các từ tấn công trực diện vào uy tín (lừa đảo, fake).
- **Legal/Health Terms (Quan trọng nhất)** — Các từ liên quan đến pháp luật/sức khỏe (ngộ độc, công an). Nhóm này có `weight` (trọng số) cao nhất (15). Chỉ cần xuất hiện 1–2 bài chứa từ này là phải báo ngay, không cần đợi viral.
- **Slang Terms** — Từ lóng giới trẻ hay dùng.

### 3. Volume Trigger (Kích hoạt theo Lượng thảo luận)

Đây là phần "thông minh" nhất của cấu hình:

- **Level WARNING (Cảnh báo sớm)** — Tăng 50% so với trung bình 7 ngày → _Báo cho team trực page để ý._
- **Level CRITICAL (Báo động đỏ)** — Tăng 150% chỉ trong 1 giờ → _Báo ngay cho Giám đốc Marketing._
- **`baseline`** — So sánh với trung bình 7 ngày trước cùng giờ đó (để tránh báo ảo vào giờ cao điểm buổi tối).

### 4. Sentiment Trigger (Kích hoạt theo Cảm xúc & ABSA)

- **`min_sample_size`: 50** — Tham số **cực kỳ quan trọng**.
  - _Tại sao?_ Nếu chỉ có 2 người comment, 1 người chê → Tỷ lệ tiêu cực là 50%. Hệ thống sẽ báo động giả liên tục.
  - _Giải pháp:_ Hệ thống chỉ bắt đầu tính toán tỷ lệ khi có ít nhất 50 thảo luận.
- **`absa_aspect_alert`** — Tận dụng tính năng ABSA:
  - Nếu người dùng chê "Giá đắt" → Có thể bỏ qua.
  - Nhưng nếu chê "Vệ sinh" (`hygiene`) hoặc "An toàn" (`safety`) → Ngưỡng kích hoạt chỉ cần 15% tiêu cực là phải báo ngay. Đây là tư duy của Senior Marketer.

### 5. Influencer Trigger (Kích hoạt theo Người ảnh hưởng)

Chia làm 2 loại rủi ro:

- **Macro Influencer** — Người nổi tiếng (KOLs) nói xấu → Độ phủ rộng (Reach).
- **Viral Post** — Người thường nhưng bài viết được Share nhiều → Độ lan truyền sâu (Virality). Chỉ số Share quan trọng hơn Like trong khủng hoảng.

---

## Tổng kết

File cấu hình JSON này giải quyết được bài toán:

1. **Không báo động giả** — Nhờ `min_sample_size` và `baseline`.
2. **Không bỏ lọt** — Nhờ chia nhóm từ khóa và phát hiện Viral Post từ người dùng thường.
3. **Hành động cụ thể** — Phân loại được đâu là _Cảnh báo_ (để theo dõi) và đâu là _Khủng hoảng_ (để xử lý ngay).

> Sử dụng cấu trúc này cho đồ án sẽ giúp bạn ghi điểm tuyệt đối về tư duy hệ thống và nghiệp vụ Marketing.

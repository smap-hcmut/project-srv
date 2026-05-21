# Crisis Config Runtime Mapping

Tài liệu này mô tả rõ phần nào trong cấu hình Brand Risk Monitoring của project đang thật sự ảnh hưởng đến runtime crisis alert hiện tại.

## Nguyên tắc hiện tại

`analysis-srv` chưa đánh crisis trực tiếp từ toàn bộ raw comment hoặc toàn bộ field trong `project crisis config`. Runtime hiện tại đọc các báo cáo BI đã chuẩn hóa, sau đó tính ba tín hiệu chính:

| Runtime signal | Nguồn dữ liệu runtime | Field cấu hình đang có tác dụng | Field đang lưu để mở rộng |
| --- | --- | --- | --- |
| `issue_pressure` | `top_issues_report.issues[0].issue_pressure_proxy` | `volume_trigger.rules[].threshold_percent_growth` | `volume_trigger.metric`, `comparison_window_hours`, `baseline` |
| `sentiment_collapse` | Proxy từ `sov_report.entities`, đếm entity có `delta_mention_count < 0` | `sentiment_trigger.rules[type=NEGATIVE_SPIKE].threshold_percent` | `min_sample_size`, `ASPECT_NEGATIVE`, aspect list |
| `controversy_spike` | `thread_controversy_report.threads[0].controversy_score_proxy` | `influencer_trigger.rules[type=VIRAL_NEGATIVE].min_comments` | `HIGH_REACH`, follower/share threshold, required sentiment, AND/OR logic |
| Runtime action | Crisis level sau khi scoring | `response_policy.adaptive_crawl`, `response_policy.notification` | Không có |

## Field reserved

`keywords_trigger` đang được lưu như taxonomy ngôn ngữ nghiệp vụ cho marketing user: nhóm rủi ro như lỗi dịch vụ, COD, an toàn, vận hành, chăm sóc khách hàng. Trong runtime hiện tại, field này chưa phải là điều kiện trực tiếp để phát `CRISIS_ALERT`.

Điều đó có nghĩa là nếu alert ghi `Affected Aspects: controversy_spike`, nguyên nhân trực tiếp đến từ điểm tranh luận của thread trong BI report, không phải do keyword group như `Service failure` hoặc `Payment and COD` match trực tiếp.

## Cách giải thích cho business/user

Brand Risk Monitoring hiện tại có hai lớp:

1. **Runtime-active controls**: các ngưỡng đang ảnh hưởng trực tiếp đến crisis scoring và hành động hệ thống.
2. **Reserved business taxonomy**: các nhóm keyword/aspect giúp chuẩn hóa ngôn ngữ marketing, preset theo domain, và là nền tảng để mở rộng scorer keyword-driven sau này.

Trên UI, các section nên được gắn nhãn:

| Nhãn UI | Ý nghĩa |
| --- | --- |
| `Runtime active` | Field trong section này đang ảnh hưởng trực tiếp đến hệ thống. |
| `Partly active` | Chỉ một vài field trong section đang được dùng; field còn lại được lưu để mở rộng. |
| `Reserved` | Field được lưu và có giá trị nghiệp vụ, nhưng chưa phải điều kiện trực tiếp của crisis runtime. |

## Hướng mở rộng sau bảo vệ

Nếu cần biến toàn bộ cấu hình thành runtime thật, thứ tự ưu tiên nên là:

1. Gắn `keywords_trigger` vào scorer bằng cách match keyword group trên comment/post evidence đã chuẩn hóa.
2. Gắn `min_sample_size` để tránh phát alert khi sample quá nhỏ.
3. Gắn `ASPECT_NEGATIVE` với mart-level aspect sentiment.
4. Gắn `HIGH_REACH` khi pipeline có author reach/follower/share đáng tin cậy.
5. Thêm historical baseline thật cho `comparison_window_hours` và `baseline`.

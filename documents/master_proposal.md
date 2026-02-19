# MASTER PROPOSAL: Project Service — Centralized Management \u0026 Orchestration

**Phiên bản:** 1.0
**Ngày tạo:** 19/02/2026
**Repo:** project-srv (Go)
**Tác giả:** Auto-generated based on `documents/gap_analysis.md` \u0026 `documents/service_interactions.md`

---

## 1. MỤC TIÊU TỔNG THỂ

Xây dựng **Project Service** trở thành "bộ não" quản lý trung tâm của hệ thống SMAP Enterprise, chịu trách nhiệm:

1. **Centralized Management:** Quản lý vòng đời (Lifecycle) của `Campaign` và `Project`.
2. **Data Source Orchestration:** Cấu hình và điều phối việc thu thập dữ liệu từ đa nguồn (Facebook, TikTok, YouTube, Webhook, File Upload) thông qua `ingest-srv`.
3. **Crisis Configuration:** Thiết lập và quản lý các quy tắc phát hiện khủng hoảng (Crisis Detection Rules) để `analytics-srv` thực thi.
4. **Integration Hub:** Kết nối `Frontend` (Web UI) với các services backend khác (`ingest`, `analytics`, `knowledge`, `noti`).

---

## 2. PHÂN TÍCH GAP: HIỆN TẠI vs MỤC TIÊU

Dựa trên [Gap Analysis](gap_analysis.md), dưới đây là trạng thái hiện tại so với mục tiêu thiết kế.

### 2.1 Entity Management

| Khía cạnh | Hiện tại | Mục tiêu |
| :--- | :--- | :--- |
| **Campaign** | ✅ Đã implement CRUD, Status enum | Quản lý Campaign (Group Projects) |
| **Project** | ✅ Đã implement CRUD, Status enum | Quản lý Entity Monitoring Unit |
| **Data Source** | ❌ **Thiếu hoàn toàn** quản lý Data Source | Tầng 1 của Entity Hierarchy: Quản lý sources (FB, TikTok, File...). Cần module `internal/data_source`. |
| **Wizard Flow** | ❌ Chỉ có basic CRUD | Multi-step Wizard (6 bước): Info → Sources → Analytics Config → Onboarding → Dry Run → Activate. |

### 2.2 Integration \u0026 Orchestration

| Khía cạnh | Hiện tại | Mục tiêu |
| :--- | :--- | :--- |
| **Ingest Integ.** | ❌ Chưa có connection | Trigger Dry Run, quản lý Onboarding (AI Schema Mapping) thông qua API `ingest-srv`. |
| **Kafka Events** | ❌ Chưa implement Producer | Publish events lifecycle: `project.created`, `project.activated`, `project.dryrun.requested`. |
| **Analytics Cfg** | ❌ Chưa có config endpoint | Endpoint `PUT /projects/{id}/analytics-config` để bật/tắt module (Sentiment, Aspect...). |
| **Dry Run** | ❌ Chưa có flow | Async Dry Run flow: Request → Kafka → Ingest (Worker) → Result → Polling. |

### 2.3 Dashboard \u0026 Insights

| Khía cạnh | Hiện tại | Mục tiêu |
| :--- | :--- | :--- |
| **Dashboard API** | ❌ Chưa có API `GET /dashboard` | Aggregation API trả về summary, time-series, top keywords (proxy call sang `analytics-srv`). |
| **Crisis UI** | ✅ Config Validation (Crisis Rules) | Hiển thị Alert Log và quản lý trạng thái Crisis (Real-time update từ `noti-srv`/`analytics-srv`). |

---

## 3. KẾ HOẠCH THỰC THI CHI TIẾT

Lộ trình phát triển được ưu tiên để giải quyết các BLOCKER (Data Source) trước.

### Phase 1: Core Data Source (Nền tảng)

**Mục tiêu:** Implement tầng quản lý Data Source (Physical Data Unit).

* **Module:** `internal/data_source`
* **Database:** Tạo bảng `ingest.data_sources`, `ingest.dryrun_results`.
* **API:** CRUD cho các loại source (`FILE_UPLOAD`, `WEBHOOK`, `FACEBOOK`, `TIKTOK`, `YOUTUBE`).
* **Integration:** Min/Max validation cho config của từng loại source.

### Phase 2: AI Schema Agent (Data Onboarding)

**Mục tiêu:** Hỗ trợ mapping schema tự động cho `FILE_UPLOAD` và `WEBHOOK`.

* **Flow:** Upload Sample → LLM Analyze → Suggest Mapping → User Confirm.
* **Integration:** Gọi `ingest-srv` (hoặc trực tiếp LLM SDK nếu logic đơn giản) để phân tích file.

### Phase 3: Dry Run Mechanism (Async Validation)

**Mục tiêu:** Kiểm tra kết nối và dữ liệu mẫu trước khi activate project.

* **Flow:** Trigger `POST /dryrun` → Kafka `project.dryrun.requested` → Ingest Worker xử lý → Cập nhật `dryrun_results`.
* **API:** Polling endpoint `GET /dryrun/status`.

### Phase 4: Activation \u0026 Wizard Completion

**Mục tiêu:** Hoàn thiện flow kích hoạt Project.

* **Logic:** Validate rules (Onboarding done? Dry run success?) → Chuyển trạng thái `ACTIVE`.
* **Event:** Publish `project.activated` để `ingest-srv` bắt đầu scheduler crawl thật.

### Phase 5: Dashboard \u0026 Integration

**Mục tiêu:** Hiển thị số liệu.

* **API:** `GET /projects/{id}/dashboard`.
* **Client:** Implement HTTP Client gọi `analytics-srv` lấy insights.

---

## 4. INFRASTRUCTURE CHANGES

### 4.1 Queue/Topic Topology (Kafka)

| Topic | Producer | Consumer | Format | Mục đích |
| :--- | :--- | :--- | :--- | :--- |
| `project.created` | **project-srv** | analytics, knowledge | CloudEvents JSON | Khởi tạo context cho các service khác. |
| `project.activated` | **project-srv** | ingest-worker | CloudEvents JSON | Trigger cron jobs thu thập dữ liệu. |
| `project.dryrun.requested` | **project-srv** | ingest-worker | CloudEvents JSON | Yêu cầu chạy thử (test connection/crawl). |
| `project.crisis.started` | **project-srv** | ingest-worker, noti | CloudEvents JSON | Reactive Scaling (tăng tần suất crawl) & Alert. |

### 4.2 Database Schema (PostgreSQL)

Ref: [Schema Document](schema.md) \u0026 [Gap Analysis](gap_analysis.md)

* **Schema:** `schema_project` (hiện tại), cần thêm `ingest` schema.
* **New Tables (Ingest):**
  * `ingest.data_sources`: Lưu cấu hình nguồn dữ liệu.
  * `ingest.dryrun_results`: Lưu kết quả chạy thử.
* **New Columns:**
  * `analytics_config` (JSONB) trong bảng `projects`.

---

## 5. FILE CHANGES SUMMARY

### 5.1 Đã Implement (Current Repository Status)

* **Entities:** `internal/campaign`, `internal/project`, `internal/crisis` (Logic CRUD cơ bản).
* **Middleware:** `internal/middleware` (Auth, Logging).
* **Config:** `config/` (Basic structure).
* **Docs:** `documents/gap_analysis.md`, `documents/schema.md`, `documents/service_interactions.md`.

### 5.2 Files Cần Tạo (Execution)

| File/Module | Mục đích |
| :--- | :--- |
| `internal/data_source/*` | Module quản lý Data Source (CRUD, Onboarding). |
| `internal/kafka/producer.go` | Implement Kafka Producer để gửi events. |
| `internal/client/*.go` | HTTP Clients gọi `ingest`, `analytics`, `knowledge`. |
| `internal/project/usecase/activate.go`| Logic kích hoạt project phức tạp. |
| `migration/xxx_create_data_sources.sql` | Migration script cho Data Source tables. |

---

## 6. DATA FLOW (HIGH LEVEL)

Tham chiếu chi tiết: [Service Interactions](service_interactions.md)

1. **Project Setup Logic:**
    * User (UI) → **Project Srv** (Create Project/Source) → DB (Pending).
    * User (UI) → **Project Srv** (Dry Run) → **Kafka** → **Ingest Worker** (Crawl Test) → DB (Result).
    * User (UI) → **Project Srv** (Activate) → **Kafka** → **Ingest Worker** (Start Schedule).

2. **Monitoring Logic:**
    * User (UI) → **Project Srv** (View Dashboard) → **Analytics Srv** (Get Metrics) → Return UI.
    * **Analytics Srv** (Detect Crisis) → **Project Srv** (Update Status) → **Kafka** (Reactive Scale).

---

## 7. TIMELINE \u0026 PHÂN BỔ (DỰ KIẾN)

* **Week 1:** Phase 1 (Data Source Module) + Phase 3 (Kafka Setup).
* **Week 2:** Phase 2 (AI Schema/Onboarding) + Phase 3 (Dry Run Logic).
* **Week 3:** Phase 4 (Activation Logic) + Phase 5 (Dashboard Proxy).
* **Week 4:** Testing \u0026 Integration Review.

---

## 8. RISKS \u0026 MITIGATIONS

* **Kafka Availability:** Nếu Kafka down, flow Dry Run/Activation bị gián đoạn. → **Mitigation:** Implement Retry logic và Outbox pattern (nếu cần consistency cao).
* **Ingest Sync:** `ingest-srv` thay đổi API dry run/onboarding. → **Mitigation:** Định nghĩa rõ Shared Contract (OpenAPI/gRPC proto) giữa 2 team.
* **Data Source Complexity:** Mỗi loại source (FB, TikTok, File) có config khác nhau. → **Mitigation:** Sử dụng JSONB cho field `config` thay vì fixed columns, validate bằng Go structs flexible.

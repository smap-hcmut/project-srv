# Design: Refactor LLM Keyword - Runtime Graceful Degradation

## Context

LLM service được tích hợp để check ambiguity của keywords. Hiện tại:
- Config validation khi start service hoạt động đúng (bắt buộc `LLM_API_KEY`)
- Nhưng khi runtime Gemini API lỗi → request bị ảnh hưởng
- Config chưa sync giữa các môi trường

## Goals / Non-Goals

**Goals:**
- Runtime graceful degradation: Gemini lỗi → tiếp tục dùng keyword gốc, không block
- Config sync giữa tất cả config files
- Project creation luôn thành công (200 OK) nếu data hợp lệ, bất kể LLM status

**Non-Goals:**
- Thay đổi startup validation (vẫn bắt buộc `LLM_API_KEY`)
- Thay đổi LLM provider logic
- Thêm retry mechanism mới

## Decisions

### Decision 1: Runtime Error = Log Warning + Continue

Khi LLM call fail ở runtime, validator sẽ:
1. Log warning với error details
2. **Tiếp tục validation với keyword gốc** (không modify, không reject)
3. Return keyword bình thường để lưu DB

```go
// validator.go - current behavior (đã đúng, chỉ cần verify)
isAmbiguous, context, err := uc.llmProvider.CheckAmbiguity(ctx, normalized)
if err != nil {
    // Log warning but DO NOT return error - continue with original keyword
    uc.l.Warnf(ctx, "LLM ambiguity check failed for '%s': %v", normalized, err)
    // Continue - keyword is still valid, just without ambiguity check
} else if isAmbiguous {
    uc.l.Warnf(ctx, "Keyword '%s' might be ambiguous. Context: %s", normalized, context)
}
return normalized, nil  // Always return the keyword
```

**Rationale:** LLM check là enhancement, không phải core validation. Keyword đã pass basic validation (length, charset, stopwords) thì vẫn hợp lệ.

### Decision 2: Keep Startup Validation

Giữ nguyên validation trong `httpserver/new.go`:
```go
if srv.llmConfig.APIKey == "" {
    return errors.New("LLM API key is required")
}
```

**Rationale:** Config phải đúng để service start. Nếu thiếu API key thì admin cần biết ngay.

### Decision 3: Config Sync

Đồng bộ các biến LLM giữa:
- `template.env` - development template
- `k8s/configmap.yaml` - non-sensitive config
- `k8s/secret.yaml.template` - sensitive config (API key)

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Ambiguous keywords không được detect khi LLM lỗi | Log warning để admin monitor |
| Silent failures | Structured logging với level WARN, có thể setup alert |

## Migration Plan

1. Verify `validator.go` đã handle error đúng (log + continue)
2. Sync config files
3. Test với Gemini API lỗi (invalid key, timeout)
4. Verify project creation vẫn 200 OK

**Rollback:** Không cần - chỉ là behavior clarification và config sync.

## Open Questions

None - behavior đã rõ ràng.

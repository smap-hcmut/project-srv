# Change: Refactor LLM Keyword - Runtime Graceful Degradation & Config Sync

## Why

Hiện tại LLM keyword validation có 2 vấn đề:

1. **Config không sync**: Các biến môi trường LLM chưa được đồng bộ giữa `template.env`, `k8s/configmap.yaml`, `k8s/secret.yaml.template` và source code
2. **Runtime error handling**: Khi Gemini API call lỗi lúc runtime (404, timeout, invalid response...), request bị ảnh hưởng - trong khi LLM check chỉ là tính năng phụ trợ, nên fallback gracefully và tiếp tục dùng keyword gốc

**Lưu ý**: Validation `LLM_API_KEY` khi start service vẫn giữ nguyên - config phải đúng để service start.

## What Changes

### Config Sync
- Đồng bộ tất cả LLM config variables giữa các file config
- Thêm comment documentation cho từng biến

### Runtime Graceful Degradation
- Khi Gemini API call lỗi (timeout, 404, invalid response, network error...) → log warning và **tiếp tục dùng keyword gốc**
- Project creation PHẢI trả 200 OK và lưu keyword vào DB bình thường ngay cả khi LLM check fail
- LLM ambiguity check chỉ là "nice to have" - nếu lỗi thì bỏ qua, không block flow

### Không thay đổi
- Vẫn giữ validation `LLM_API_KEY` khi start service (bắt buộc config đúng)
- Logic LLM provider và retry mechanism giữ nguyên

## Impact

- Affected specs: `llm-service`, `keyword-validation`
- Affected code:
  - `project/internal/keyword/usecase/validator.go` - đảm bảo LLM error không block, tiếp tục với keyword gốc
  - `project/template.env` - sync config
  - `project/k8s/configmap.yaml` - sync config  
  - `project/k8s/secret.yaml.template` - sync config

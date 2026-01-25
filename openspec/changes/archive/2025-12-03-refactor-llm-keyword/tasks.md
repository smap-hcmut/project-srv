# Implementation Tasks

## 1. Verify & Fix Runtime Error Handling
- [x] 1.1 Review `project/internal/keyword/usecase/validator.go` - verify LLM errors are logged but not returned
- [x] 1.2 Ensure `validateOne()` always returns `(normalized, nil)` even when LLM fails
- [x] 1.3 Add/update test case for LLM failure scenario - verify keyword is still returned

## 2. Sync Configuration Files
- [x] 2.1 Sync `.env` với `template.env` - đảm bảo `.env` có đủ các biến LLM cần thiết
- [x] 2.2 Update `project/template.env` - add comments, ensure all LLM vars present with correct defaults
- [x] 2.3 Update `project/k8s/configmap.yaml` - sync non-sensitive LLM config vars (đã có đủ)
- [x] 2.4 Update `project/k8s/secret.yaml.template` - ensure `LLM_API_KEY` placeholder present (đã có)
- [x] 2.5 Verify `config/config.go` matches all env vars (đã verify)
- [x] 2.6 Fix LLM_MODEL từ `gemini-1.5-flash` sang `gemini-2.0-flash` (model cũ đã deprecated)

## 3. Run Service với Env Thật
- [x] 3.1 Run `make run-api` với `.env` thật
- [x] 3.2 Test API endpoint tạo project với Cookie auth header
- [x] 3.3 Verify service start thành công với LLM config

## 4. Integration Testing - Gemini Success Cases
- [x] 4.1 Test Gemini API trực tiếp - verify API key và model hoạt động
- [x] 4.2 Test project creation với ambiguous keyword "Apple" - verify warning logged
- [x] 4.3 Test project creation với non-ambiguous keyword "VinFast" - verify no warning
- [x] 4.4 Test project creation với competitor keyword "Samsung" - verify ambiguity detected
- [x] 4.5 Verify 200 OK returned và keywords saved to DB

## 5. Integration Testing - Gemini Failure Cases
- [x] 5.1 Test với invalid API key (404 error) - verify 200 OK, keyword saved
- [x] 5.2 Verify logs contain appropriate warnings when LLM fails
- [ ] 5.3 Test với LLM timeout scenario (optional - cần mock)

## 6. Update Default Model Config
- [x] 6.1 Update `template.env` default model từ `gemini-1.5-flash` sang `gemini-2.0-flash`
- [x] 6.2 Update `k8s/configmap.yaml` default model
- [x] 6.3 Update `config/config.go` envDefault cho LLM_MODEL

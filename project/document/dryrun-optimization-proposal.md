# Đề xuất tối ưu logic Dry-run

## Vấn đề hiện tại

### 1. Tốc độ xử lý không đồng đều

#### Thực tế từ crawler logs:

- **1 keyword**: ~16 giây (tesla: 08:30:58 → 08:31:14)
- **2 keywords**: ~32 giây (tesla + vinfast: 08:30:58 → 08:31:30)
- **Tính toán scale**:
  - 10 keywords: ~160 giây (2.7 phút) → **VỰT QUÁ 90s WebSocket timeout**
  - 50 keywords: ~800 giây (13.3 phút) → **KHÔNG KHẢ THI**

#### Breakdown thời gian per keyword:

- **Search phase**: ~7-9 giây (tìm videos)
- **Scraping phase**: ~5-7 giây (scrape 3 videos + comments + creators)
- **Total per keyword**: ~16 giây average

#### Vấn đề nghiêm trọng:

- **Project lớn**: 50 keywords → 800 giây → **WebSocket timeout sau 90s**
- **Project nhỏ**: 5 keywords → 80 giây → **Gần timeout limit**
- **Chênh lệch**: Lên đến 50x về thời gian xử lý

### 2. Resource không tối ưu

- Crawler phải xử lý toàn bộ keywords array
- Không có cơ chế sample/limit ở business layer
- Technical limit (3 items/keyword) không giải quyết được vấn đề số lượng keywords

### 3. WebSocket Connection Constraints

- **Client behavior**: Sau khi call dry-run, client tạo WebSocket connection tới WS service
- **Timeout limit**: WebSocket chỉ duy trì trong **90 giây**
- **Data streaming**: Dry-run results được pub qua Redis và stream qua WebSocket
- **Critical risk**: Nếu dry-run > 90s → client mất connection → user không nhận được results
- **User experience**: Client phải chờ trong uncertainty nếu process quá lâu

## Đề xuất giải pháp

### 1. Nguyên tắc thiết kế

#### Separation of Concerns

```
Project Service (Business Layer)
├── Quyết định: Bao nhiêu keywords cho dry-run?
├── Sample/limit keywords theo business rules
├── Áp dụng sampling strategy phù hợp
└── Gửi request đã optimize xuống Collector

Collector Service (Technical Layer)
├── Nhận request đã được optimize
├── Enforce technical limits (LimitPerKeyword = 3)
└── Forward tới Crawler
```

### 2. Chiến lược Sampling Keywords

#### Option 1: Fixed Sample Size (Đề xuất chính)

- **Constant**: `DRY_RUN_KEYWORD_LIMIT = 5` (dựa trên thực tế 16s/keyword)
- **Logic**: Luôn chọn tối đa 5 keywords cho dry-run
- **Benefit**: Thời gian xử lý constant (~80 giây, safe under 90s)
- **Calculation**: 5 keywords × 16s = 80s (10s buffer)

#### Option 2: Percentage-based Sampling

- **Rule**: Chọn tối đa 10% keywords, minimum 3, maximum 5
- **Logic**:
  - 50 keywords → 5 keywords (10%, capped)
  - 10 keywords → 3 keywords (30%, minimum)
  - 100 keywords → 5 keywords (5%, capped)
- **Timing**: Luôn ≤ 80 giây (5 × 16s)

#### Option 3: Tiered Sampling (Dựa trên thực tế performance)

- **Tiny project** (≤3 keywords): 100% keywords (~48s)
- **Small project** (4-5 keywords): 100% keywords (~80s)
- **Medium project** (6-15 keywords): 5 keywords (~80s)
- **Large project** (>15 keywords): 5 keywords (~80s)

### 3. Keyword Selection Strategy

#### Priority-based Selection

1. **High Priority**: Keywords có search volume cao
2. **Medium Priority**: Keywords có competition thấp
3. **Random**: Nếu không có metadata, chọn random

#### Diversity Selection

- Chọn keywords từ các category khác nhau
- Tránh chọn keywords quá giống nhau
- Đảm bảo representative sample

### 4. Implementation Strategy

#### Phase 1: Business Logic Update

- Thêm `DryRunKeywordSampler` trong Project Service
- Implement sampling algorithms
- Add configuration cho sampling parameters

#### Phase 2: Configuration Management

- Environment variables cho sampling config
- Database config cho dynamic adjustment
- Monitoring metrics cho performance tracking

#### Phase 3: Performance Monitoring

- Track dry-run execution time
- Monitor keyword distribution
- Alert nếu thời gian xử lý vượt threshold

## Chi tiết Implementation

### 1. Configuration Parameters

```yaml
# Dry-run Configuration (Based on real performance: 16s/keyword)
DRY_RUN_KEYWORD_LIMIT: 5 # Max keywords cho dry-run (5 × 16s = 80s)
DRY_RUN_SAMPLING_STRATEGY: "fixed" # fixed|percentage|tiered
DRY_RUN_MIN_KEYWORDS: 1 # Minimum keywords
DRY_RUN_MAX_KEYWORDS: 5 # Maximum keywords (hard limit for WebSocket)
DRY_RUN_TIMEOUT: 85 # Timeout in seconds (5s buffer)
DRY_RUN_WS_TIMEOUT: 90 # WebSocket connection timeout
DRY_RUN_BUFFER_TIME: 10 # Safety buffer before WebSocket timeout
DRY_RUN_KEYWORD_TIME_ESTIMATE: 16 # Average seconds per keyword
```

### 2. Sampling Algorithm Flow

```
Input: Project với N keywords
↓
Apply Sampling Strategy
├── Fixed: Chọn min(N, LIMIT) keywords
├── Percentage: Chọn min(N × %, MAX) keywords
└── Tiered: Chọn theo tier rules
↓
Apply Selection Strategy
├── Priority-based: Sort by priority → take top K
├── Random: Shuffle → take first K
└── Diversity: Group by category → sample from each
↓
Output: Optimized keyword list (≤ LIMIT)
```

### 3. Performance Targets

#### Thời gian xử lý mục tiêu (Dựa trên thực tế 16s/keyword)

- **Dry-run time**: ≤ 80 seconds (5 keywords × 16s)
- **Target time**: 48-80 seconds (3-5 keywords range)
- **Variance**: ≤ ±5 seconds giữa các project (do fixed keyword count)
- **Throughput**: Có thể xử lý multiple dry-run concurrent
- **WebSocket compatibility**: Đảm bảo hoàn thành trước 90s timeout với 10s buffer

#### Resource utilization

- **CPU**: Không vượt quá 70% trong dry-run
- **Memory**: Stable memory usage
- **Network**: Optimized request batching

## Benefits của đề xuất

### 1. Predictable Performance

- Thời gian dry-run constant (~30-45s)
- Resource usage predictable
- Better user experience

### 2. Scalability

- Không bị bottleneck bởi project size
- Có thể handle concurrent dry-runs
- Easy to tune performance parameters

### 3. Business Value

- Faster feedback cho users
- Consistent experience across projects
- Better resource utilization

### 4. Maintainability

- Clear separation of concerns
- Configurable parameters
- Easy to test và monitor

## Migration Plan

### Phase 1: Preparation (1 week)

- Design sampling interfaces
- Create configuration structure
- Write unit tests cho sampling logic

### Phase 2: Implementation (2 weeks)

- Implement sampling algorithms
- Update Project Service logic
- Add monitoring metrics

### Phase 3: Testing & Rollout (1 week)

- Performance testing
- A/B testing với old logic
- Gradual rollout với feature flag

### Phase 4: Optimization (ongoing)

- Monitor performance metrics
- Fine-tune sampling parameters
- Collect user feedback

## Risk Mitigation

### 1. Sampling Accuracy

- **Risk**: Sample không representative
- **Mitigation**: Multiple sampling strategies, validation metrics

### 2. Performance Regression

- **Risk**: New logic slower than expected
- **Mitigation**: Performance benchmarks, rollback plan

### 3. User Experience

- **Risk**: Users expect full keyword analysis
- **Mitigation**: Clear communication, option để run full analysis

### 4. WebSocket Timeout Risk

- **Risk**: Dry-run vượt quá 90s → WebSocket disconnect → user mất data
- **Mitigation**:
  - Hard limit dry-run ≤ 75s
  - Fallback mechanism nếu WebSocket timeout
  - Progress indicators cho user
  - Retry mechanism với smaller sample size

## Success Metrics

### 1. Performance Metrics (Dựa trên thực tế crawler performance)

- Dry-run execution time variance < 10% (do fixed keyword count)
- 95th percentile response time < 80s (5 keywords × 16s)
- 99th percentile response time < 85s (với network delays)
- WebSocket connection success rate > 99.5%
- Average time per keyword: ~16 seconds (monitoring target)
- Resource utilization stable

### 2. Business Metrics

- User satisfaction với dry-run speed
- Adoption rate của dry-run feature
- Conversion rate từ dry-run sang full run

### 3. Technical Metrics

- System stability during peak usage
- Error rate < 1%
- Concurrent dry-run capacity

## WebSocket Integration Considerations

### 1. Client-Server Flow

```
Client Request Dry-run
↓
Server starts processing (optimized keywords)
↓
Client establishes WebSocket connection
↓
Server publishes results to Redis
↓
WebSocket service streams data to client
↓
Connection must complete within 90 seconds
```

### 2. Timeout Management Strategy (Dựa trên real performance data)

- **Processing timeout**: 85 seconds (hard limit, 5s buffer)
- **WebSocket timeout**: 90 seconds (client-side)
- **Safety buffer**: 10 seconds (realistic buffer)
- **Expected time**: 48-80 seconds (3-5 keywords × 16s/keyword)
- **Fallback**: Nếu timeout, return partial results
- **Emergency fallback**: Nếu > 70s, reduce to 3 keywords max

### 3. Real-time Progress Updates

- Stream progress updates qua WebSocket
- Show keyword processing status
- Estimated completion time
- Graceful handling nếu connection lost

## Conclusion

Đề xuất này sẽ giải quyết vấn đề tốc độ không đồng đều của dry-run bằng cách:

1. **Standardize** số lượng keywords được xử lý
2. **Optimize** business logic ở đúng layer
3. **Maintain** quality của sample results
4. **Ensure** WebSocket compatibility với 90s timeout
5. **Improve** overall user experience với predictable timing

Kết quả là một hệ thống dry-run có performance predictable, scalable, maintainable và **WebSocket-compatible**.

## Appendix: Real Performance Analysis

### Crawler Performance Data (From Logs)

```
Keyword "tesla": 08:30:58 → 08:31:14 = 16 seconds
Keyword "vinfast": 08:31:14 → 08:31:30 = 16 seconds
Total 2 keywords: 32 seconds
Average: 16 seconds per keyword
```

### Performance Breakdown per Keyword:

1. **Search phase**: 7-9 seconds

   - TikTok search API call
   - Video URL extraction
   - Strategy execution

2. **Scraping phase**: 5-7 seconds

   - Video metadata scraping (3 videos)
   - Comments scraping (fast mode)
   - Creator profile scraping

3. **Processing overhead**: 1-2 seconds
   - Data mapping
   - Redis publishing
   - Result formatting

### Scaling Implications:

- **Current logic**: N keywords × 16s = total time
- **WebSocket constraint**: Must complete in 90s
- **Maximum safe keywords**: 90s ÷ 16s = 5.6 → **5 keywords max**
- **Recommended limit**: 5 keywords (80s + 10s buffer)

### Risk Assessment:

- **High risk**: >5 keywords (>80s, near timeout)
- **Medium risk**: 4-5 keywords (64-80s, acceptable)
- **Low risk**: 1-3 keywords (16-48s, very safe)
- **Optimal**: 3-4 keywords (48-64s, good balance)

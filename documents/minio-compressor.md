# MinIO Compression & Async Upload - Usage Guide

## Overview

Hướng dẫn sử dụng tính năng nén và upload async cho MinIO trong SMAP Identity Service.

## Use Cases

### Use Case 1: Upload JSON Data với Compression và Async

**Scenario**: Bạn có JSON data lớn (ví dụ: danh sách users, transactions) và muốn:
- Nén để tiết kiệm storage (giảm 60-70% dung lượng)
- Upload async để không block request
- Track progress real-time

**Flow**:
```
JSON Data → Compress (Zstd) → Async Upload → MinIO
                                    ↓
                            Progress Tracking
                                    ↓
                            Get File ID (Object Name)
```

### Use Case 2: Download và Decode JSON từ File ID

**Scenario**: Bạn có file ID (object name) và muốn:
- Download file từ MinIO
- Auto-decompress nếu file đã được nén
- Parse thành JSON object

**Flow**:
```
File ID → Download from MinIO → Auto-Decompress → JSON Data
```

---

## Implementation Guide

### 1. Upload JSON Data với Compression và Async

#### Step 1: Prepare JSON Data

```go
package main

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "log"
    "time"
    
    "smap-project/pkg/minio"
    "smap-project/config"
)

// Example: User batch data
type UserBatch struct {
    Users []User `json:"users"`
    Total int    `json:"total"`
}

type User struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    Email    string `json:"email"`
    Metadata map[string]interface{} `json:"metadata"`
}

func main() {
    // 1. Prepare your JSON data
    batch := UserBatch{
        Users: []User{
            {ID: "1", Name: "John Doe", Email: "john@example.com"},
            {ID: "2", Name: "Jane Smith", Email: "jane@example.com"},
            // ... thousands of users
        },
        Total: 10000,
    }
    
    // 2. Marshal to JSON
    jsonData, err := json.Marshal(batch)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Original JSON size: %d bytes\n", len(jsonData))
}
```

#### Step 2: Initialize MinIO Client

```go
// Load configuration
cfg, err := config.Load()
if err != nil {
    log.Fatal(err)
}

// Connect to MinIO (singleton pattern)
ctx := context.Background()
minioClient, err := minio.Connect(ctx, cfg.MinIO)
if err != nil {
    log.Fatal(err)
}

fmt.Println("Connected to MinIO successfully")
```

#### Step 3: Upload với Compression và Async

```go
// Generate unique object name
objectName := fmt.Sprintf("user-batches/%s/%s.json.zst", 
    time.Now().Format("2006/01/02"),
    uuid.New().String())

// Create upload request
uploadReq := &minio.UploadRequest{
    BucketName:   cfg.MinIO.Bucket,
    ObjectName:   objectName,
    OriginalName: "user-batch.json",
    Reader:       bytes.NewReader(jsonData),
    Size:         int64(len(jsonData)),
    ContentType:  "application/json",
    
    // IMPORTANT: Enable compression
    EnableCompression: true,
    CompressionLevel:  2, // 0=none, 1=fastest, 2=default, 3=best
    
    // Optional: Add metadata
    Metadata: map[string]string{
        "batch-id":    "batch-001",
        "created-by":  "system",
        "record-count": fmt.Sprintf("%d", batch.Total),
    },
}

// Upload async
taskID, err := minioClient.UploadAsync(ctx, uploadReq)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Upload queued successfully!\n")
fmt.Printf("Task ID: %s\n", taskID)
fmt.Printf("Object Name: %s\n", objectName)
```

#### Step 4: Track Upload Progress

```go
// Option A: Poll for progress
ticker := time.NewTicker(500 * time.Millisecond)
defer ticker.Stop()

for {
    select {
    case <-ticker.C:
        status, err := minioClient.GetUploadStatus(taskID)
        if err != nil {
            log.Printf("Error getting status: %v", err)
            continue
        }
        
        fmt.Printf("\rUploading: %.2f%% (%d/%d bytes) - Status: %s",
            status.Percentage,
            status.BytesUploaded,
            status.TotalBytes,
            status.Status)
        
        if status.Status == minio.UploadStatusCompleted {
            fmt.Println("\nUpload completed!")
            
            // Get result
            result, _ := minioClient.WaitForUpload(taskID, 1*time.Second)
            if result.FileInfo != nil {
                fmt.Printf("Compression Stats:\n")
                fmt.Printf("   Original size: %d bytes\n", result.FileInfo.UncompressedSize)
                fmt.Printf("   Compressed size: %d bytes\n", result.FileInfo.CompressedSize)
                fmt.Printf("   Compression ratio: %.2f%%\n", result.FileInfo.CompressionRatio*100)
                fmt.Printf("   Saved: %.2f%%\n", (1-result.FileInfo.CompressionRatio)*100)
            }
            return
        }
        
        if status.Status == minio.UploadStatusFailed {
            fmt.Printf("\nUpload failed: %s\n", status.Error)
            return
        }
    }
}

// Option B: Wait for completion (blocking)
result, err := minioClient.WaitForUpload(taskID, 30*time.Second)
if err != nil {
    log.Fatal(err)
}

if result.Error != nil {
    log.Fatalf("Upload failed: %v", result.Error)
}

fmt.Printf("Upload completed in %v\n", result.Duration)
```

#### Complete Upload Example

```go
func UploadJSONBatch(ctx context.Context, minioClient minio.MinIO, data interface{}) (string, error) {
    // 1. Marshal to JSON
    jsonData, err := json.Marshal(data)
    if err != nil {
        return "", fmt.Errorf("failed to marshal JSON: %w", err)
    }
    
    // 2. Generate object name
    objectName := fmt.Sprintf("batches/%s/%s.json.zst",
        time.Now().Format("2006/01/02"),
        uuid.New().String())
    
    // 3. Create upload request
    uploadReq := &minio.UploadRequest{
        BucketName:        "data-batches",
        ObjectName:        objectName,
        OriginalName:      "batch.json",
        Reader:            bytes.NewReader(jsonData),
        Size:              int64(len(jsonData)),
        ContentType:       "application/json",
        EnableCompression: true,
        CompressionLevel:  2,
    }
    
    // 4. Upload async
    taskID, err := minioClient.UploadAsync(ctx, uploadReq)
    if err != nil {
        return "", fmt.Errorf("failed to queue upload: %w", err)
    }
    
    // 5. Wait for completion
    result, err := minioClient.WaitForUpload(taskID, 30*time.Second)
    if err != nil {
        return "", fmt.Errorf("upload timeout: %w", err)
    }
    
    if result.Error != nil {
        return "", fmt.Errorf("upload failed: %w", result.Error)
    }
    
    // 6. Return object name (file ID)
    return objectName, nil
}

// Usage
fileID, err := UploadJSONBatch(ctx, minioClient, userBatch)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("File uploaded successfully!\n")
fmt.Printf("File ID: %s\n", fileID)
// Save this fileID to database or return to client
```

---

### 2. Download và Decode JSON từ File ID

#### Step 1: Download File với Auto-Decompression

```go
func DownloadAndDecodeJSON(ctx context.Context, minioClient minio.MinIO, fileID string, target interface{}) error {
    // 1. Create download request
    downloadReq := &minio.DownloadRequest{
        BucketName: "data-batches",
        ObjectName: fileID, // This is the file ID from upload
    }
    
    // 2. Download file (auto-decompresses if compressed)
    reader, headers, err := minioClient.DownloadFile(ctx, downloadReq)
    if err != nil {
        return fmt.Errorf("failed to download file: %w", err)
    }
    defer reader.Close()
    
    fmt.Printf("Downloading: %s\n", headers.ContentType)
    fmt.Printf("Content-Length: %s bytes\n", headers.ContentLength)
    
    // 3. Read all data (already decompressed)
    data, err := io.ReadAll(reader)
    if err != nil {
        return fmt.Errorf("failed to read file: %w", err)
    }
    
    fmt.Printf("Downloaded %d bytes (decompressed)\n", len(data))
    
    // 4. Unmarshal JSON
    if err := json.Unmarshal(data, target); err != nil {
        return fmt.Errorf("failed to unmarshal JSON: %w", err)
    }
    
    return nil
}

// Usage
var batch UserBatch
err := DownloadAndDecodeJSON(ctx, minioClient, fileID, &batch)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("JSON decoded successfully!\n")
fmt.Printf("Total users: %d\n", batch.Total)
fmt.Printf("First user: %s (%s)\n", batch.Users[0].Name, batch.Users[0].Email)
```

#### Step 2: Get File Info (Optional)

```go
// Get file metadata before downloading
fileInfo, err := minioClient.GetFileInfo(ctx, "data-batches", fileID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("File Info:\n")
fmt.Printf("   Name: %s\n", fileInfo.OriginalName)
fmt.Printf("   Size: %d bytes\n", fileInfo.Size)
fmt.Printf("   Type: %s\n", fileInfo.ContentType)
fmt.Printf("   Compressed: %v\n", fileInfo.IsCompressed)

if fileInfo.IsCompressed {
    fmt.Printf("   Compressed size: %d bytes\n", fileInfo.CompressedSize)
    fmt.Printf("   Uncompressed size: %d bytes\n", fileInfo.UncompressedSize)
    fmt.Printf("   Compression ratio: %.2f%%\n", fileInfo.CompressionRatio*100)
}

fmt.Printf("   Last modified: %s\n", fileInfo.LastModified)
fmt.Printf("   ETag: %s\n", fileInfo.ETag)

// Check metadata
if batchID, ok := fileInfo.Metadata["batch-id"]; ok {
    fmt.Printf("   Batch ID: %s\n", batchID)
}
```

#### Complete Download Example

```go
func GetUserBatchByID(ctx context.Context, minioClient minio.MinIO, fileID string) (*UserBatch, error) {
    // 1. Download and decode
    var batch UserBatch
    
    downloadReq := &minio.DownloadRequest{
        BucketName: "data-batches",
        ObjectName: fileID,
    }
    
    reader, _, err := minioClient.DownloadFile(ctx, downloadReq)
    if err != nil {
        return nil, fmt.Errorf("download failed: %w", err)
    }
    defer reader.Close()
    
    // 2. Read decompressed data
    data, err := io.ReadAll(reader)
    if err != nil {
        return nil, fmt.Errorf("read failed: %w", err)
    }
    
    // 3. Unmarshal JSON
    if err := json.Unmarshal(data, &batch); err != nil {
        return nil, fmt.Errorf("unmarshal failed: %w", err)
    }
    
    return &batch, nil
}

// Usage
batch, err := GetUserBatchByID(ctx, minioClient, fileID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Retrieved batch with %d users\n", batch.Total)
```

---

## Complete Example: Full Workflow

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "time"
    
    "smap-project/config"
    "smap-project/pkg/minio"
)

type Transaction struct {
    ID        string  `json:"id"`
    Amount    float64 `json:"amount"`
    UserID    string  `json:"user_id"`
    Timestamp int64   `json:"timestamp"`
}

type TransactionBatch struct {
    Transactions []Transaction `json:"transactions"`
    Count        int           `json:"count"`
}

func main() {
    // 1. Setup
    cfg, _ := config.Load()
    ctx := context.Background()
    minioClient, _ := minio.Connect(ctx, cfg.MinIO)
    
    // 2. Create sample data
    batch := TransactionBatch{
        Transactions: make([]Transaction, 10000),
        Count:        10000,
    }
    
    for i := 0; i < 10000; i++ {
        batch.Transactions[i] = Transaction{
            ID:        fmt.Sprintf("txn-%d", i),
            Amount:    float64(i * 100),
            UserID:    fmt.Sprintf("user-%d", i%100),
            Timestamp: time.Now().Unix(),
        }
    }
    
    // 3. Upload with compression
    fmt.Println("Starting upload...")
    fileID, err := uploadBatch(ctx, minioClient, batch)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Upload complete! File ID: %s\n\n", fileID)
    
    // 4. Wait a moment
    time.Sleep(1 * time.Second)
    
    // 5. Download and decode
    fmt.Println("Starting download...")
    retrieved, err := downloadBatch(ctx, minioClient, fileID)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Download complete! Retrieved %d transactions\n", retrieved.Count)
    fmt.Printf("First transaction: %+v\n", retrieved.Transactions[0])
}

func uploadBatch(ctx context.Context, client minio.MinIO, batch TransactionBatch) (string, error) {
    // Marshal to JSON
    jsonData, err := json.Marshal(batch)
    if err != nil {
        return "", err
    }
    
    fmt.Printf("Original size: %d bytes\n", len(jsonData))
    
    // Generate object name
    objectName := fmt.Sprintf("transactions/%s/%s.json.zst",
        time.Now().Format("2006-01-02"),
        time.Now().Format("150405"))
    
    // Upload async with compression
    uploadReq := &minio.UploadRequest{
        BucketName:        "transaction-batches",
        ObjectName:        objectName,
        OriginalName:      "transactions.json",
        Reader:            bytes.NewReader(jsonData),
        Size:              int64(len(jsonData)),
        ContentType:       "application/json",
        EnableCompression: true,
        CompressionLevel:  2,
        Metadata: map[string]string{
            "batch-count": fmt.Sprintf("%d", batch.Count),
            "created-at":  time.Now().Format(time.RFC3339),
        },
    }
    
    taskID, err := client.UploadAsync(ctx, uploadReq)
    if err != nil {
        return "", err
    }
    
    // Wait for completion
    result, err := client.WaitForUpload(taskID, 30*time.Second)
    if err != nil {
        return "", err
    }
    
    if result.Error != nil {
        return "", result.Error
    }
    
    // Print compression stats
    if result.FileInfo != nil {
        fmt.Printf("Compression stats:\n")
        fmt.Printf("   Uncompressed: %d bytes\n", result.FileInfo.UncompressedSize)
        fmt.Printf("   Compressed: %d bytes\n", result.FileInfo.CompressedSize)
        fmt.Printf("   Ratio: %.2f%%\n", result.FileInfo.CompressionRatio*100)
        fmt.Printf("   Saved: %.2f%%\n", (1-result.FileInfo.CompressionRatio)*100)
        fmt.Printf("   Duration: %v\n", result.Duration)
    }
    
    return objectName, nil
}

func downloadBatch(ctx context.Context, client minio.MinIO, fileID string) (*TransactionBatch, error) {
    // Download with auto-decompression
    downloadReq := &minio.DownloadRequest{
        BucketName: "transaction-batches",
        ObjectName: fileID,
    }
    
    reader, _, err := client.DownloadFile(ctx, downloadReq)
    if err != nil {
        return nil, err
    }
    defer reader.Close()
    
    // Read decompressed data
    data, err := io.ReadAll(reader)
    if err != nil {
        return nil, err
    }
    
    fmt.Printf("Downloaded size: %d bytes (decompressed)\n", len(data))
    
    // Unmarshal JSON
    var batch TransactionBatch
    if err := json.Unmarshal(data, &batch); err != nil {
        return nil, err
    }
    
    return &batch, nil
}
```

**Expected Output:**
```
Starting upload...
Original size: 890000 bytes
Compression stats:
   Uncompressed: 890000 bytes
   Compressed: 45000 bytes
   Ratio: 5.06%
   Saved: 94.94%
   Duration: 1.2s
Upload complete! File ID: transactions/2024-11-25/193916.json.zst

Starting download...
Downloaded size: 890000 bytes (decompressed)
Download complete! Retrieved 10000 transactions
First transaction: {ID:txn-0 Amount:0 UserID:user-0 Timestamp:1732534756}
```

---

## Key Points

### Upload Flow
1. **Prepare JSON** → Marshal to bytes
2. **Create UploadRequest** → Set `EnableCompression: true`
3. **Upload Async** → Get task ID
4. **Track Progress** → Poll status or wait for completion
5. **Get File ID** → Save object name for later retrieval

### Download Flow
1. **Use File ID** → Object name from upload
2. **Download File** → Auto-decompresses if compressed
3. **Read Data** → Already decompressed
4. **Unmarshal JSON** → Parse to struct

### Important Notes
- Compression is **automatic** when `EnableCompression: true`
- Decompression is **automatic** on download
- File ID = Object Name (e.g., `batches/2024-11-25/uuid.json.zst`)
- Compression works best for JSON/text (60-95% reduction)
- Async upload doesn't block your request
- Progress tracking available in real-time

---

## Best Practices

### 1. Object Naming Convention
```go
// Good: Organized by date
"batches/2024/11/25/uuid.json.zst"
"transactions/2024-11-25/batch-001.json.zst"

// Bad: Flat structure
"batch-uuid.json.zst"
```

### 2. Compression Strategy
```go
// Compress JSON/text data
EnableCompression: true,
CompressionLevel: 2,  // Default is good for most cases

// Don't compress already compressed data
// (images, videos, archives)
EnableCompression: false,
```

### 3. Error Handling
```go
// Always check upload result
result, err := client.WaitForUpload(taskID, 30*time.Second)
if err != nil {
    return fmt.Errorf("upload timeout: %w", err)
}

if result.Error != nil {
    return fmt.Errorf("upload failed: %w", result.Error)
}
```

### 4. Metadata Usage
```go
// Add useful metadata for tracking
Metadata: map[string]string{
    "batch-id":     batchID,
    "record-count": fmt.Sprintf("%d", count),
    "created-by":   userID,
    "created-at":   time.Now().Format(time.RFC3339),
    "schema-version": "v1.0",
},
```

---

## API Reference

### Upload Methods
```go
// Sync upload (blocking)
fileInfo, err := client.UploadFile(ctx, uploadReq)

// Async upload (non-blocking)
taskID, err := client.UploadAsync(ctx, uploadReq)

// Get upload status
status, err := client.GetUploadStatus(taskID)

// Wait for upload completion
result, err := client.WaitForUpload(taskID, timeout)

// Cancel upload
err := client.CancelUpload(taskID)
```

### Download Methods
```go
// Download file (auto-decompresses)
reader, headers, err := client.DownloadFile(ctx, downloadReq)

// Stream file (for large files)
reader, headers, err := client.StreamFile(ctx, downloadReq)

// Get file info
fileInfo, err := client.GetFileInfo(ctx, bucketName, objectName)
```

### Configuration
```bash
# Environment variables
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=default-bucket
MINIO_ASYNC_UPLOAD_WORKERS=4
MINIO_ASYNC_UPLOAD_QUEUE_SIZE=100
```

---

## Summary

Với implementation này, bạn có thể:
- Upload JSON data với compression (giảm 60-95% dung lượng)
- Upload async để không block request
- Track progress real-time
- Download và auto-decompress
- Parse JSON trực tiếp sau download

**Performance:**
- Compression: 60-95% size reduction
- Upload: Non-blocking, 50-100x faster response
- Download: Auto-decompression, transparent to user

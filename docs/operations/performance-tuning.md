# Performance Tuning Guide

> **Note**: This document is a work in progress. Contributions are welcome!

This guide covers performance optimization strategies for go-whatsapp-web-multidevice in production environments.

## Table of Contents

- [Overview](#overview)
- [Resource Requirements](#resource-requirements)
- [Database Optimization](#database-optimization)
- [Media Processing](#media-processing)
- [Concurrency and Scaling](#concurrency-and-scaling)
- [Network Optimization](#network-optimization)
- [Benchmarking](#benchmarking)

## Overview

Key performance factors:

- **Database**: SQLite vs PostgreSQL, connection pooling
- **Media Processing**: FFmpeg optimization, compression settings
- **Memory**: Chat storage, media caching
- **Concurrency**: Goroutine management, connection limits
- **Network**: Webhook delivery, WhatsApp protocol

## Resource Requirements

### Minimum Requirements

- **CPU**: 1 core
- **Memory**: 512MB RAM
- **Disk**: 1GB (database + media cache)
- **Network**: Stable internet connection

### Recommended for Production

- **CPU**: 2 cores
- **Memory**: 1GB RAM (2GB for high volume)
- **Disk**: 10GB SSD (faster I/O)
- **Network**: Low latency, high bandwidth

### Resource Usage by Volume

| Messages/Day | CPU | Memory | Disk I/O |
|--------------|-----|--------|----------|
| < 1,000 | 0.5 core | 512MB | Low |
| 1,000-10,000 | 1 core | 1GB | Medium |
| 10,000-100,000 | 2 cores | 2GB | High |
| > 100,000 | 4+ cores | 4GB+ | Very High |

## Database Optimization

### SQLite (Default)

**Pros**:
- Simple, no separate server
- Good for low-medium volume
- File-based, easy backups

**Cons**:
- Limited concurrency
- Not ideal for high volume
- Single machine only

**Optimization**:

```bash
# Use WAL mode for better concurrency
sqlite3 storages/whatsapp.db "PRAGMA journal_mode=WAL;"

# Increase cache size
sqlite3 storages/whatsapp.db "PRAGMA cache_size=10000;"

# Optimize on startup
sqlite3 storages/whatsapp.db "PRAGMA optimize;"
```

**Environment Variables**:

```bash
DB_URI=file:storages/whatsapp.db?cache=shared&_journal_mode=WAL
```

### PostgreSQL (Recommended for Production)

**Pros**:
- Better concurrency
- Scalable to high volume
- Replication support
- Advanced features

**Configuration**:

```bash
DB_URI=postgres://user:password@localhost:5432/whatsapp?sslmode=disable
```

**PostgreSQL Settings** (`postgresql.conf`):

```conf
# Increase connection pool
max_connections = 100

# Memory settings
shared_buffers = 256MB
effective_cache_size = 1GB
work_mem = 16MB
maintenance_work_mem = 128MB

# Write-Ahead Log
wal_buffers = 16MB
checkpoint_completion_target = 0.9

# Query planning
random_page_cost = 1.1  # For SSD
effective_io_concurrency = 200  # For SSD
```

### Connection Pooling

> **TODO**: Implement configurable connection pool settings

Recommended settings (when implemented):

```bash
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m
```

### Chat Storage

Chat storage can consume significant disk space.

**Disable if not needed**:

```bash
WHATSAPP_CHAT_STORAGE=false
```

**Periodic Cleanup**:

```sql
-- Delete messages older than 30 days
DELETE FROM messages WHERE timestamp < datetime('now', '-30 days');

-- Vacuum to reclaim space
VACUUM;
```

**Automated Cleanup** (cron job):

```bash
# Add to crontab
0 2 * * * sqlite3 /app/storages/chatstorage.db "DELETE FROM messages WHERE timestamp < datetime('now', '-30 days'); VACUUM;"
```

## Media Processing

Media processing is CPU-intensive. Optimization is critical for high volume.

### FFmpeg Optimization

See [Audio Format Optimization](audio-optimization.md) for detailed FFmpeg tuning.

**Key Settings**:

1. **Hardware Acceleration**:

```bash
# Check available encoders
ffmpeg -encoders | grep h264

# Use hardware encoder (if available)
ffmpeg -hwaccel auto -i input.mp4 -c:v h264_nvenc output.mp4
```

2. **Compression Presets**:

```bash
# Fast encoding (lower quality)
ffmpeg -i input.mp4 -preset ultrafast -crf 28 output.mp4

# Balanced (default)
ffmpeg -i input.mp4 -preset medium -crf 23 output.mp4

# High quality (slower)
ffmpeg -i input.mp4 -preset slow -crf 18 output.mp4
```

3. **Parallel Processing**:

```bash
# Use multiple threads
ffmpeg -threads 4 -i input.mp4 output.mp4

# Auto-detect CPU cores
ffmpeg -threads 0 -i input.mp4 output.mp4
```

### Media File Limits

Configure appropriate limits:

```bash
WHATSAPP_MAX_FILE_SIZE_IMAGE=20971520    # 20MB
WHATSAPP_MAX_FILE_SIZE_VIDEO=104857600   # 100MB
WHATSAPP_MAX_FILE_SIZE_DOCUMENT=52428800 # 50MB
```

### Media Cleanup

Clean up old media files periodically:

```bash
# Delete media files older than 7 days
find src/statics/media -type f -mtime +7 -delete

# Clean up empty directories
find src/statics/media -type d -empty -delete
```

**Automated Cleanup** (cron job):

```bash
# Add to crontab
0 3 * * * find /app/statics/media -type f -mtime +7 -delete
```

### Disable Compression (if needed)

For trusted sources or pre-compressed media:

> **TODO**: Add configuration option to disable compression

## Concurrency and Scaling

### Goroutine Management

Go handles concurrency well, but beware of goroutine leaks.

**Current Async Operations**:
- Webhook delivery
- Media processing
- Auto-reconnect checking

**Best Practices**:
- Use contexts for cancellation
- Monitor goroutine count
- Set timeouts on operations

### Webhook Concurrency

Webhooks are delivered concurrently with retry logic.

**Current Behavior**:
- Each webhook URL gets its own goroutine
- 5 retry attempts with exponential backoff
- Timeout: 30 seconds per attempt

**Tuning** (future):

```bash
WEBHOOK_CONCURRENT_DELIVERIES=10
WEBHOOK_RETRY_ATTEMPTS=5
WEBHOOK_TIMEOUT=30s
```

### Rate Limiting

> **TODO**: Implement rate limiting

WhatsApp has rate limits. Respect them to avoid blocks:

- Messages: ~50-100 per minute per number
- Media: Lower limits due to upload time
- Group operations: Conservative limits

Planned rate limiting:

```bash
RATE_LIMIT_REQUESTS_PER_MINUTE=60
RATE_LIMIT_BURST=10
```

### Horizontal Scaling

**Current Limitation**: One WhatsApp account per instance.

**Options for Scaling**:

1. **Multiple Instances** (different WhatsApp numbers):
   - Use Admin API mode
   - Each instance = one WhatsApp account
   - Load balancer routes by account

2. **Shared Session Storage** (future):
   - PostgreSQL for session
   - Multiple readers, single writer
   - Complex, requires careful implementation

## Network Optimization

### Connection Settings

**WhatsApp Connection**:
- Uses WebSocket over TLS
- Auto-reconnect enabled by default
- Keep-alive pings

**HTTP Client**:

> **TODO**: Make HTTP client settings configurable

Recommended settings:

```go
http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
}
```

### Webhook Optimization

**Reduce Latency**:
- Host webhook endpoint close to WhatsApp server
- Use CDN for static webhook responses
- Optimize endpoint processing time

**Increase Reliability**:
- Implement idempotency on webhook endpoint
- Return 200 OK quickly, process async
- Handle duplicate deliveries gracefully

### DNS Caching

Enable DNS caching to reduce lookup latency:

```bash
# On Linux, install nscd
sudo apt-get install nscd
sudo systemctl enable nscd
sudo systemctl start nscd
```

## Benchmarking

### Application Benchmarks

> **TODO**: Create benchmark suite

Planned benchmarks:

```go
func BenchmarkSendTextMessage(b *testing.B) {
    for i := 0; i < b.N; i++ {
        SendTextMessage(ctx, "6281234567890", "Hello")
    }
}

func BenchmarkMediaCompression(b *testing.B) {
    for i := 0; i < b.N; i++ {
        CompressImage("test.jpg")
    }
}
```

Run benchmarks:

```bash
cd src && go test -bench=. -benchmem ./...
```

### Load Testing

> **TODO**: Add load testing examples

Tools:
- **k6**: Modern load testing
- **Apache Bench**: Simple HTTP testing
- **Locust**: Python-based load testing

Example with k6:

```javascript
import http from 'k6/http';

export default function() {
  http.post('http://localhost:3000/send/text', JSON.stringify({
    phone: '6281234567890',
    message: 'Load test message'
  }), {
    headers: { 'Content-Type': 'application/json' },
  });
}
```

Run:

```bash
k6 run --vus 10 --duration 30s loadtest.js
```

### Profiling

Go has excellent profiling tools.

**CPU Profiling**:

```bash
# Add to code temporarily
import _ "net/http/pprof"
go http.ListenAndServe("localhost:6060", nil)

# Capture profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
```

**Memory Profiling**:

```bash
go tool pprof http://localhost:6060/debug/pprof/heap
```

**Goroutine Profiling**:

```bash
curl http://localhost:6060/debug/pprof/goroutine?debug=2
```

## Monitoring Performance

See [Monitoring Guide](monitoring.md) for detailed monitoring setup.

**Key Metrics to Monitor**:
- Request latency (P50, P95, P99)
- Throughput (requests/second)
- Error rate
- CPU usage
- Memory usage
- Database query time
- Media processing time

## Optimization Checklist

### Database
- [ ] Use PostgreSQL for production
- [ ] Enable WAL mode (SQLite)
- [ ] Configure connection pooling
- [ ] Add database indexes
- [ ] Periodic vacuum/cleanup
- [ ] Disable chat storage if not needed

### Media
- [ ] Configure FFmpeg for hardware acceleration
- [ ] Set appropriate compression presets
- [ ] Implement media cleanup cron job
- [ ] Set reasonable file size limits
- [ ] Use SSD for media storage

### Application
- [ ] Enable production mode (disable debug)
- [ ] Configure rate limiting
- [ ] Optimize webhook delivery
- [ ] Set appropriate timeouts
- [ ] Monitor goroutine count

### Infrastructure
- [ ] Use SSD storage
- [ ] Allocate sufficient RAM
- [ ] Enable DNS caching
- [ ] Use CDN for static assets (if any)
- [ ] Optimize network latency

## Troubleshooting Performance Issues

### High CPU Usage

**Symptoms**: CPU consistently > 80%

**Possible Causes**:
- Heavy media processing
- Too many concurrent operations
- Inefficient code

**Solutions**:
1. Profile CPU usage: `go tool pprof`
2. Reduce media compression quality
3. Implement rate limiting
4. Optimize hot code paths

### High Memory Usage

**Symptoms**: Memory usage growing over time

**Possible Causes**:
- Memory leak
- Chat storage accumulating
- Media files in memory
- Goroutine leak

**Solutions**:
1. Profile memory: `go tool pprof heap`
2. Disable chat storage
3. Clean up old media files
4. Check for goroutine leaks
5. Restart periodically (temporary fix)

### Slow Database Queries

**Symptoms**: High database query latency

**Possible Causes**:
- Missing indexes
- Large tables
- SQLite concurrency limits
- Slow disk

**Solutions**:
1. Add database indexes
2. Switch to PostgreSQL
3. Use SSD storage
4. Vacuum database
5. Archive old data

### Slow Media Processing

**Symptoms**: Long processing time for media

**Possible Causes**:
- Large file sizes
- Slow compression
- CPU limitations
- No hardware acceleration

**Solutions**:
1. Enable FFmpeg hardware acceleration
2. Use faster compression preset
3. Reduce compression quality
4. Limit file sizes
5. Add more CPU cores

## Best Practices

1. **Start Simple**: Use defaults, optimize only when needed
2. **Measure First**: Profile before optimizing
3. **Incremental Changes**: Change one thing at a time
4. **Monitor**: Always monitor production performance
5. **Document**: Document all performance tuning changes
6. **Test**: Load test before deploying optimizations

## Related Documentation

- [Audio Format Optimization](audio-optimization.md) - FFmpeg tuning
- [Monitoring Guide](monitoring.md) - Performance monitoring
- [Deployment Guides](../guides/deployment/) - Production deployment
- [Architecture Overview](../developer/architecture.md) - System design

## Contributing

This guide is incomplete. Contributions needed:

- Add benchmark suite
- Document load testing procedures
- Add profiling examples
- Share production optimization experiences
- Add scaling strategies
- Document performance metrics

See [Contributing Guide](../developer/contributing.md) for details.

## Resources

- [Go Performance](https://dave.cheney.net/high-performance-go-workshop/dotgo-paris.html) - Optimization guide
- [Go Profiling](https://golang.org/doc/diagnostics.html) - Official profiling docs
- [FFmpeg Optimization](https://trac.ffmpeg.org/wiki/Encode/HighQualityAudio) - FFmpeg guides
- [PostgreSQL Performance](https://wiki.postgresql.org/wiki/Performance_Optimization) - PostgreSQL tuning

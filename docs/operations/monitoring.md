# Monitoring Guide

> **Note**: This document is a work in progress. Contributions are welcome!

This guide covers monitoring and observability for go-whatsapp-web-multidevice in production environments.

## Table of Contents

- [Overview](#overview)
- [Health Checks](#health-checks)
- [Metrics](#metrics)
- [Logging](#logging)
- [Alerting](#alerting)
- [Troubleshooting](#troubleshooting)

## Overview

Monitoring is essential for maintaining reliable WhatsApp API services. Key areas to monitor:

- **Application Health**: Connection status, uptime
- **Performance**: Response times, throughput
- **Resources**: CPU, memory, disk usage
- **Errors**: Failed requests, WhatsApp disconnections
- **Business Metrics**: Messages sent/received, active users

## Health Checks

### REST Mode

Check application health using the `/app/devices` endpoint:

```bash
curl http://localhost:3000/app/devices
```

**Healthy Response**:
```json
{
  "code": 200,
  "message": "Success",
  "results": [
    {
      "name": "Chrome",
      "device": "Desktop",
      "platform": "linux"
    }
  ]
}
```

### MCP Mode

Check SSE endpoint:

```bash
curl http://localhost:8080/sse
```

### Docker Health Check

Add to `docker-compose.yml`:

```yaml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:3000/app/devices"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 40s
```

### Kubernetes Liveness Probe

```yaml
livenessProbe:
  httpGet:
    path: /app/devices
    port: 3000
  initialDelaySeconds: 30
  periodSeconds: 30
```

## Metrics

> **TODO**: Implement Prometheus metrics export

### Planned Metrics

#### Application Metrics
- `whatsapp_connection_status` - Connection state (1=connected, 0=disconnected)
- `whatsapp_uptime_seconds` - Application uptime
- `whatsapp_reconnect_total` - Total reconnection attempts

#### Message Metrics
- `whatsapp_messages_sent_total` - Messages sent (by type)
- `whatsapp_messages_received_total` - Messages received (by type)
- `whatsapp_messages_failed_total` - Failed message sends

#### Performance Metrics
- `whatsapp_request_duration_seconds` - HTTP request duration histogram
- `whatsapp_webhook_delivery_duration_seconds` - Webhook delivery time
- `whatsapp_media_processing_duration_seconds` - Media processing time

#### Resource Metrics
- `process_cpu_usage_percent` - CPU usage
- `process_memory_usage_bytes` - Memory usage
- `process_open_fds` - Open file descriptors

### Example Prometheus Configuration

```yaml
scrape_configs:
  - job_name: 'whatsapp'
    static_configs:
      - targets: ['localhost:3000']
    metrics_path: '/metrics'  # TODO: Implement
    scrape_interval: 15s
```

## Logging

### Log Levels

- `INFO` - Normal operation
- `WARN` - Warning conditions
- `ERROR` - Error conditions
- `DEBUG` - Detailed debugging (use with `--debug=true`)

### Enable Debug Logging

```bash
./whatsapp rest --debug=true
```

Or via environment variable:

```bash
APP_DEBUG=true ./whatsapp rest
```

### Log Format

Application uses structured logging with logrus:

```
INFO[2025-01-14T10:30:00Z] Message sent successfully  phone=6281234567890 type=text
```

### Collecting Logs

#### Docker Logs

```bash
docker logs -f whatsapp
docker logs --since 1h whatsapp
docker logs --tail 100 whatsapp
```

#### Systemd Logs

```bash
journalctl -u whatsapp-rest -f
journalctl -u whatsapp-rest --since "1 hour ago"
```

### Centralized Logging

#### ELK Stack (Elasticsearch, Logstash, Kibana)

```yaml
# docker-compose.yml
services:
  whatsapp:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  filebeat:
    image: docker.elastic.co/beats/filebeat:8.0.0
    volumes:
      - /var/lib/docker/containers:/var/lib/docker/containers:ro
      - ./filebeat.yml:/usr/share/filebeat/filebeat.yml:ro
```

#### Loki + Grafana

```yaml
# docker-compose.yml
services:
  whatsapp:
    logging:
      driver: loki
      options:
        loki-url: "http://localhost:3100/loki/api/v1/push"
```

## Alerting

> **TODO**: Set up alerting examples

### Critical Alerts

1. **Service Down**
   - Condition: Health check fails for 3 consecutive checks
   - Action: Page on-call engineer

2. **WhatsApp Disconnected**
   - Condition: Connection status = disconnected for > 5 minutes
   - Action: Trigger auto-reconnect, notify team

3. **High Error Rate**
   - Condition: Error rate > 10% over 5 minutes
   - Action: Notify team

4. **Memory Usage High**
   - Condition: Memory usage > 90% for 10 minutes
   - Action: Alert and investigate

### Warning Alerts

1. **Message Send Failures**
   - Condition: Failed messages > 50 in 10 minutes
   - Action: Notify team

2. **Webhook Delivery Failures**
   - Condition: Webhook failures > 20% over 15 minutes
   - Action: Check webhook endpoint

3. **Disk Space Low**
   - Condition: Disk usage > 85%
   - Action: Clean up old media files

### Example AlertManager Configuration

```yaml
# TODO: Implement Prometheus metrics first
route:
  group_by: ['alertname', 'instance']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'team-notifications'

receivers:
  - name: 'team-notifications'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/YOUR/WEBHOOK/URL'
        channel: '#whatsapp-alerts'
```

## Troubleshooting

### High Memory Usage

**Symptoms**: Memory continuously increases

**Possible Causes**:
- Media files not being cleaned up
- Chat storage accumulating
- Memory leak in application

**Solutions**:
1. Clean old media files from `src/statics/media/`
2. Truncate chat storage database
3. Restart application
4. Monitor memory usage over time

### Connection Drops

**Symptoms**: Frequent WhatsApp disconnections

**Possible Causes**:
- Network instability
- WhatsApp rate limiting
- Session expired

**Solutions**:
1. Check network connectivity
2. Enable auto-reconnect (enabled by default)
3. Re-authenticate if session expired
4. Check WhatsApp account for blocks/restrictions

### Webhook Delivery Failures

**Symptoms**: Webhooks not reaching destination

**Possible Causes**:
- Webhook endpoint down
- Network issues
- HMAC signature mismatch
- Endpoint timeout

**Solutions**:
1. Verify webhook URL is accessible
2. Check HMAC secret matches on both sides
3. Increase endpoint timeout
4. Check endpoint logs for errors

## Dashboards

> **TODO**: Create example Grafana dashboards

### Planned Dashboard Panels

1. **Overview**
   - Connection status indicator
   - Uptime percentage
   - Messages sent/received counters
   - Current error rate

2. **Performance**
   - Request duration (P50, P95, P99)
   - Webhook delivery time
   - Media processing time
   - Throughput (requests/second)

3. **Resources**
   - CPU usage graph
   - Memory usage graph
   - Disk usage graph
   - Network I/O

4. **Errors**
   - Error rate over time
   - Failed messages by type
   - Webhook failures
   - Recent error log entries

## Best Practices

1. **Monitor Connection Status**: Set up alerts for disconnections
2. **Track Error Rates**: Monitor for spikes in errors
3. **Log Aggregation**: Centralize logs for analysis
4. **Regular Backups**: Backup database and session files
5. **Capacity Planning**: Monitor resource trends
6. **Runbooks**: Document common issues and resolutions
7. **On-Call Procedures**: Have escalation paths defined

## Related Documentation

- [Performance Tuning](performance-tuning.md) - Optimization strategies
- [Security Best Practices](security-best-practices.md) - Security guidelines
- [Deployment Guides](../guides/deployment/) - Deployment methods
- [Troubleshooting](../reference/troubleshooting.md) - Common issues

## Contributing

This document is incomplete. Contributions are welcome:

- Add Prometheus metrics implementation
- Create Grafana dashboard examples
- Add alerting rule examples
- Document monitoring tools integration
- Share production monitoring experience

See [Contributing Guide](../developer/contributing.md) for details.

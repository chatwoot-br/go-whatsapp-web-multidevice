# Operations Guide

Resources for deploying, monitoring, and maintaining production systems.

## Performance and Optimization

- **[Audio Format Optimization](audio-optimization.md)** - Audio conversion and FFmpeg configuration
- **[Performance Tuning](performance-tuning.md)** - Optimization best practices

## Security

- **[Security Best Practices](security-best-practices.md)** - Security guidelines and recommendations

## Monitoring and Observability

- **[Monitoring Guide](monitoring.md)** - Metrics, logging, and alerting

## Production Considerations

### Resource Requirements

- **Memory**: 512MB minimum, 1GB recommended
- **CPU**: 1 core minimum, 2 cores recommended
- **Storage**: 1GB for database and media cache
- **FFmpeg**: Required for media processing

### Database Options

- **SQLite** (default): Simple, file-based
- **PostgreSQL**: Recommended for production

### High Availability

- Use PostgreSQL for persistent storage
- Configure auto-reconnection monitoring
- Set up health check endpoints
- Implement webhook retry logic

## Related Documentation

- **[Deployment Guides](../guides/deployment/)** - Deployment methods
- **[Configuration Reference](../reference/configuration.md)** - All configuration options
- **[Troubleshooting](../reference/troubleshooting.md)** - Common issues

---

**Version**: Compatible with v7.10.1+
**Last Updated**: 2025-12-05

# Issue Investigations

This directory contains detailed technical investigations of significant issues affecting the go-whatsapp-web-multidevice service. Unlike postmortems (which document resolved incidents), these documents track ongoing investigations and proposed solutions.

## Purpose

- Document root cause analysis for complex issues
- Provide detailed technical context for contributors
- Track proposed solutions and implementation plans
- Serve as reference for similar future issues

## Issue Index

| ID | Title | Severity | Status | Date |
|----|-------|----------|--------|------|
| [001](001-lid-jid-chat-split.md) | Conversation Split (@lid vs @s.whatsapp.net) | Critical | Open | 2025-12-19 |

## Issue Lifecycle

1. **Open** - Investigation in progress or complete, awaiting implementation
2. **In Progress** - Fix being implemented
3. **Testing** - Fix deployed to staging, under validation
4. **Resolved** - Fix deployed to production, issue closed
5. **Closed** - Moved to postmortems if significant, or archived

## Template

When creating a new issue investigation:

1. Use the naming convention: `NNN-short-description.md`
2. Include sections:
   - Issue Summary
   - Impact Assessment
   - Root Cause Analysis
   - Affected Code Paths
   - Proposed Solutions
   - Testing Plan
   - Action Items

## Related Documentation

- [Postmortems](../postmortems/) - Resolved incidents with lessons learned
- [Developer Guide](../developer/) - Contributing guidelines
- [Architecture](../developer/architecture.md) - System design reference

# ISSUE-001: Message Loss Due to Profile Picture Fetch Panic

**Issue Type**: Critical Bug
**Status**: ✅ RESOLVED (Gateway Side) / ⏳ PENDING (Chatwoot Side)
**Affected Component**: WhatsApp Web Provider Integration
**Severity**: High (message loss in production)
**Date Reported**: 2025-10-07
**Date Resolved (Gateway)**: 2025-10-07

## Summary

Users are not receiving some WhatsApp messages in Chatwoot due to a cascading failure:
1. Go WhatsApp Web Multidevice service panics when fetching profile pictures
2. Service crashes and restarts
3. During restart, Chatwoot webhook jobs fail with connection refused
4. Messages are lost and never processed

## ✅ Resolution (Gateway Side)

**Fix Applied**: 2025-10-07

The **primary issue** (Gateway panic) has been **RESOLVED** by updating the whatsmeow library in the go-whatsapp-web-multidevice service.

### Gateway Fix Details

**Solution Implemented**: Solution 1 - Update whatsmeow Library

**Changes Made**:
```bash
# go-whatsapp-web-multidevice updated from:
go.mau.fi/whatsmeow v0.0.0-20251003111114-4479f300784e

# Updated to:
go.mau.fi/whatsmeow v0.0.0-20251005083110-4fe97da162dc
go.mau.fi/libsignal v0.2.1-0.20251004173110-6e0a3f2435ed
```

**Root Cause Fixed**:
- whatsmeow library now properly supports `*store.PrivacyToken` payload type
- Upstream commits (Oct 3, 2025):
  - PR#950: "Add PrivacyToken to GetProfilePictureInfo" by purpshell
  - "fix including privacy token in GetProfilePictureInfo" by tulir

**Verification**:
- ✅ Code compiles successfully
- ✅ All tests pass
- ✅ No breaking changes detected
- ⏳ Awaiting staging and production deployment

### Chatwoot Side - Pending Improvements

The **secondary issue** (Chatwoot resilience) remains **PENDING** but is now lower priority since the gateway no longer panics.

**Recommended Chatwoot Improvements** (Solutions 3-5):
- Solution 3: Make contact info fetch non-blocking (P1 - Should still implement)
- Solution 4: Implement retry logic with exponential backoff (P2 - Optional)
- Solution 5: Separate avatar fetch into background job (P2 - Optional)

These improvements will provide additional resilience against any future gateway issues.

## Symptoms

- **User Impact**: Messages from WhatsApp (especially group messages and reactions) not appearing in Chatwoot
- **Frequency**: Intermittent, occurs when profile picture fetch is triggered
- **Affected Message Types**: Group messages, reactions, messages requiring contact info fetch

## Root Cause Analysis

### Primary Issue: Gateway Panic on Profile Picture Fetch

The Go WhatsApp Web Multidevice service panics with:
```
panic: unsupported payload type: *store.PrivacyToken

goroutine [running]:
go.mau.fi/whatsmeow/binary.(*binaryEncoder).write(...)
go.mau.fi/whatsmeow.(*Client).GetProfilePictureInfo(...)
github.com/aldinokemal/go-whatsapp-web-multidevice/usecase.serviceUser.Avatar.func1()
```

**Root Cause**: The whatsmeow library version `v0.0.0-20251003111114-4479f300784e` does not support the `*store.PrivacyToken` payload type that WhatsApp now sends in profile picture requests.

**Location**: `/whatsapp/usecase/user.go:88` in the `Avatar` function

### Secondary Issue: Chatwoot Not Handling Connection Failures Gracefully

When the gateway restarts, Chatwoot webhook jobs fail with:
```
Errno::ECONNREFUSED: Failed to open TCP connection to gowa.woot-qfrotas.svc.cluster.local:3005
(Connection refused - connect(2) for "gowa.woot-qfrotas.svc.cluster.local" port 3005)
```

**Stack Trace**:
```
app/services/whatsapp/providers/whatsapp_web_service.rb:108:in 'contact_info'
app/models/channel/whatsapp.rb:63:in 'contact_info'
app/services/whatsapp/incoming_message_whatsapp_web_service.rb:669:in 'setup_group_contact'
app/services/whatsapp/incoming_message_whatsapp_web_service.rb:64:in 'handle_incoming_group_message'
```

**Issue**: Contact info fetch (including avatar) during group message processing causes the entire webhook job to fail if the gateway is temporarily unavailable.

## Evidence

### Chatwoot Error Logs

```json
{
  "ts": "2025-10-07T18:31:04.706Z",
  "lvl": "WARN",
  "msg": "Job raised exception",
  "job": {
    "class": "Webhooks::WhatsappEventsJob",
    "args": [{
      "phone_number": "554130898195",
      "payload": {
        "chat_id": "120363230235309595",
        "from": "554130898206:3@s.whatsapp.net in 120363230235309595@g.us",
        "reaction": {
          "message": "❤️",
          "id": "3FAAB0E5F16480E454D7"
        }
      }
    }]
  }
}
```

```
ERROR -- : WhatsApp Web: Error updating contact avatar: Failed to open TCP connection
ERROR -- : WhatsApp Web: Identifier: 554130898136@s.whatsapp.net
```

### Gateway Panic Logs

```
time="2025-10-07T19:01:36Z" level=info msg="Forwarding message event to 1 configured webhook(s)"
time="2025-10-07T19:01:36Z" level=info msg="Successfully submitted webhook on attempt 1"

panic: unsupported payload type: *store.PrivacyToken

time="2025-10-07T19:01:41Z" level=info msg="[DEBUG] Starting reconnect process..."
time="2025-10-07T19:01:42Z" level=info msg="[DEBUG] Reconnection completed - IsConnected: true, IsLoggedIn: false"
```

**Pattern**: The service successfully forwards webhook events, then panics when trying to fetch profile pictures, restarts, and repeats.

## Impact Assessment

### Severity: High

- **Data Loss**: Messages are permanently lost (not queued/retried)
- **User Experience**: Critical messages may not reach agents
- **Reliability**: Service appears unreliable to end users
- **Scope**: Affects all WhatsApp Web provider inboxes in production

### Affected Operations

**After Gateway Fix (2025-10-07)**:
1. ✅ **Working**: Message webhook delivery from gateway to Chatwoot
2. ✅ **FIXED**: Profile picture fetch in gateway (whatsmeow library updated)
3. ✅ **Working**: Contact info fetch in Chatwoot (gateway no longer crashes)
4. ✅ **Working**: Group message processing
5. ✅ **Working**: Reaction message processing

**Remaining Improvement Opportunities**:
- ⚠️ Chatwoot could still be more resilient to transient gateway failures
- ⚠️ Consider implementing Solutions 3-5 for defense-in-depth

## Proposed Solutions

### Solution 1: Update whatsmeow Library (Gateway Side) - ✅ **IMPLEMENTED**

**Priority**: P0 (Must fix)
**Status**: ✅ **COMPLETED** (2025-10-07)

Update the Go WhatsApp Web Multidevice service to use a newer version of whatsmeow that supports `*store.PrivacyToken`.

**Action Items**:
1. ✅ Check whatsmeow repository for fixes/updates related to PrivacyToken
2. ✅ Update `go.mod` to use latest stable whatsmeow version (`v0.0.0-20251005083110-4fe97da162dc`)
3. ✅ Test compilation and run test suite - all tests pass
4. ⏳ Deploy updated gateway service to staging
5. ⏳ Deploy to production

**Implementation**:
```bash
go get go.mau.fi/whatsmeow@v0.0.0-20251005083110-4fe97da162dc
go mod tidy
```

**Risks**: May introduce other breaking changes - **✅ No breaking changes detected**

**Timeline**: Immediate (same day) - **✅ COMPLETED same day**

**Tracking**: Documented in `docs/issues/ISSUE-001-PROFILE-PICTURE-PANIC.md`

### Solution 2: Add Panic Recovery in Avatar Function (Gateway Side)

**Priority**: P1 (Optional defensive measure)
**Status**: ⏳ **NOT IMPLEMENTED** - May not be necessary with upstream fix

Prevent the entire service from crashing when profile picture fetch fails.

**Note**: With whatsmeow library updated, this panic should no longer occur. This solution can be implemented as an optional defensive safeguard.

**Location**: `usecase/user.go:88` in Avatar function

**Implementation**:
```go
func (service serviceUser) Avatar(ctx context.Context, jid string, isGroup bool) (response.Avatar, error) {
    // ... existing code ...

    // Wrap GetProfilePictureInfo in goroutine with panic recovery
    go func() {
        defer func() {
            if r := recover(); r != nil {
                log.Errorf("Panic in GetProfilePictureInfo: %v", r)
                // Continue without avatar instead of crashing
            }
        }()

        // ... existing GetProfilePictureInfo call ...
    }()

    // ... rest of implementation ...
}
```

**Benefits**:
- Prevents service crashes
- Allows messages to be processed even if avatar fetch fails
- Maintains service availability

**Risks**: Low

**Timeline**: Same day

### Solution 3: Make Contact Info Fetch Non-Blocking (Chatwoot Side)

**Priority**: P1 (Should fix)

Modify Chatwoot to gracefully handle contact info fetch failures without failing the entire message processing.

**Location**: `app/services/whatsapp/incoming_message_whatsapp_web_service.rb:669`

**Current Behavior**:
```ruby
def setup_group_contact
  contact_info = @inbox.channel.contact_info(sender_jid)  # Throws on connection error
  # ... process contact ...
end
```

**Proposed Fix**:
```ruby
def setup_group_contact
  begin
    contact_info = @inbox.channel.contact_info(sender_jid)
    update_contact_with_info(contact_info) if contact_info
  rescue Errno::ECONNREFUSED, Net::OpenTimeout, Net::ReadTimeout => e
    # Log error but continue processing
    Rails.logger.warn("WhatsApp Web: Failed to fetch contact info for #{sender_jid}: #{e.message}")
    # Use basic contact info from webhook payload instead
    create_contact_from_webhook_data
  end
end

private

def create_contact_from_webhook_data
  # Extract name from webhook payload (pushname field)
  contact_name = @processed_params[:pushname] || 'Unknown'
  # Create contact without avatar
  @contact_inbox.contact.update(name: contact_name)
end
```

**Benefits**:
- Messages are not lost due to temporary gateway unavailability
- Degrades gracefully (contact created without avatar)
- Can retry avatar fetch later via background job

**Risks**: Low (contact avatar may be missing initially)

**Timeline**: 1-2 days

### Solution 4: Implement Retry Logic with Exponential Backoff (Chatwoot Side)

**Priority**: P2 (Nice to have)

Add retry mechanism for transient connection failures.

**Location**: `app/services/whatsapp/providers/whatsapp_web_service.rb:108`

**Implementation**:
```ruby
def contact_info(phone_number)
  retries = 3
  backoff = 1 # seconds

  begin
    response = HTTParty.get(
      "#{api_base_url}/user/info",
      query: { phone: ensure_jid_format(phone_number) },
      headers: api_headers,
      timeout: 10
    )

    validate_response(response)
    response.parsed_response
  rescue Errno::ECONNREFUSED, Net::OpenTimeout => e
    retries -= 1
    if retries > 0
      Rails.logger.info("WhatsApp Web: Retrying contact_info after #{backoff}s (#{retries} retries left)")
      sleep(backoff)
      backoff *= 2
      retry
    else
      raise e
    end
  end
end
```

**Benefits**:
- Handles temporary gateway unavailability
- Increases success rate for transient failures

**Risks**: Low (adds latency on failures)

**Timeline**: 2-3 days

### Solution 5: Separate Avatar Fetch into Background Job (Chatwoot Side)

**Priority**: P2 (Nice to have)

Decouple avatar fetching from critical message processing path.

**Implementation**:
1. Process message immediately without avatar
2. Enqueue background job to fetch avatar
3. Update contact avatar asynchronously

**Benefits**:
- Faster message processing
- Avatar failures don't block messages
- Can retry avatar fetch independently

**Risks**: Low

**Timeline**: 3-5 days

## Recommended Fix Priority

1. **Immediate (P0)**: ✅ **COMPLETED**
   - ✅ Solution 1: Update whatsmeow library - **IMPLEMENTED 2025-10-07**

2. **Short-term (P1)**: ⏳ **RECOMMENDED FOR CHATWOOT**
   - ⚠️ Solution 3: Make contact info fetch non-blocking - **Recommended for resilience**
   - ⏳ Deploy gateway fix to production

3. **Medium-term (P2)**: 📋 **OPTIONAL IMPROVEMENTS**
   - ℹ️ Solution 2: Add panic recovery in Avatar function (defensive measure)
   - ℹ️ Solution 4: Implement retry logic
   - ℹ️ Solution 5: Background job for avatar fetch

## Testing Strategy

### Unit Tests

**Gateway Side**:
```go
func TestAvatar_PanicRecovery(t *testing.T) {
    // Test that Avatar function doesn't panic on PrivacyToken error
    // Verify error is logged
    // Verify service continues running
}
```

**Chatwoot Side**:
```ruby
# spec/services/whatsapp/incoming_message_whatsapp_web_service_spec.rb
it 'processes message even when contact_info fails' do
  allow(inbox.channel).to receive(:contact_info).and_raise(Errno::ECONNREFUSED)

  expect {
    service.perform
  }.to change(Message, :count).by(1)

  expect(Message.last.sender.name).to eq(payload[:pushname])
end
```

### Integration Tests

1. **Simulate Gateway Restart**:
   - Stop gateway service
   - Send webhook to Chatwoot
   - Verify message is created
   - Verify contact is created without avatar
   - Restart gateway
   - Verify avatar is fetched later

2. **Profile Picture with PrivacyToken**:
   - Test with WhatsApp contact that has privacy settings enabled
   - Verify service doesn't panic
   - Verify fallback avatar handling

### Manual Testing

1. Send messages to production inbox while gateway is restarting
2. Verify messages appear in Chatwoot
3. Send reaction messages
4. Send group messages
5. Verify all message types are processed

## Monitoring & Prevention

### Metrics to Add

1. **Gateway Panic Counter**: `whatsapp_web_avatar_fetch_panics_total`
2. **Chatwoot Connection Failures**: `whatsapp_web_api_connection_failures_total{endpoint="contact_info"}`
3. **Message Processing Failures**: `whatsapp_web_message_processing_failures_total{reason="connection_refused"}`

### Alerts

```yaml
- alert: WhatsAppWebGatewayPanics
  expr: rate(whatsapp_web_avatar_fetch_panics_total[5m]) > 0
  severity: critical
  message: "WhatsApp Web gateway is panicking on avatar fetch"

- alert: WhatsAppWebMessageLoss
  expr: rate(whatsapp_web_message_processing_failures_total[5m]) > 0.01
  severity: high
  message: "WhatsApp Web messages are being lost due to processing failures"
```

### Logging Improvements

**Gateway Side**:
```go
log.WithFields(log.Fields{
    "jid": jid,
    "error": err,
    "panic_recovered": true,
}).Error("Failed to fetch profile picture - continuing without avatar")
```

**Chatwoot Side**:
```ruby
Rails.logger.error({
  context: "WhatsApp Web: Contact info fetch failed",
  phone_number: phone_number,
  error: e.class.name,
  message: e.message,
  message_id: @processed_params[:message][:id],
  fallback_used: true
}.to_json)
```

## Workarounds (Until Fixed)

### For Users

1. **No action required** - messages should eventually be delivered after implementing fixes

### For Operators

1. **Monitor Gateway Health**: Set up alerts for gateway restarts
2. **Manual Message Recovery**:
   - Check Sidekiq failed jobs queue
   - Identify failed `Webhooks::WhatsappEventsJob` jobs
   - Manually retry after gateway is stable

### For Developers

1. **Temporary Patch**: Disable avatar fetching in gateway
   ```go
   // Comment out avatar fetch temporarily
   // info, err := client.GetProfilePictureInfo(...)
   return response.Avatar{}, nil // Return empty avatar
   ```

2. **Increase Job Retry**: Configure Sidekiq to retry webhook jobs with longer delays
   ```ruby
   # app/jobs/webhooks/whatsapp_events_job.rb
   sidekiq_options retry: 5, retry_in: ->(count) { 10 * (count + 1) }
   ```

## Related Documentation

- [FEAT-004 README](./README.md)
- [Implementation Story](./implementation-story.md)
- [Deployment Guide](./deployment-guide.md)
- [Webhook Payload Documentation](./webhook-payload.md)
- [Common Pitfalls & Solutions](./README.md#common-pitfalls--solutions)

## External References

- **whatsmeow Issues**: Search for "PrivacyToken" issues
- **Go WhatsApp Web Multidevice**: Check for updates and similar issues
- **WhatsApp Protocol Changes**: PrivacyToken likely related to recent WhatsApp privacy feature

## Next Steps

### Gateway Side (go-whatsapp-web-multidevice)
- [x] Check whatsmeow repository for PrivacyToken support - **COMPLETED 2025-10-07**
- [x] Update whatsmeow library (Solution 1) - **COMPLETED 2025-10-07**
- [x] Verify no breaking changes - **COMPLETED 2025-10-07**
- [x] Create issue documentation - **COMPLETED 2025-10-07**
- [ ] Test in staging environment
- [ ] Deploy to production
- [ ] Monitor for 24-48 hours to verify fix
- [ ] Optional: Implement panic recovery in Avatar function (Solution 2) as defensive measure
- [ ] Document fix in changelog

### Chatwoot Side (Optional Improvements)
- [ ] Consider implementing Solution 3: Make contact info fetch non-blocking (P1 - Recommended)
- [ ] Optional: Implement Solution 4: Retry logic with exponential backoff (P2)
- [ ] Optional: Implement Solution 5: Background job for avatar fetch (P2)
- [ ] Optional: Add monitoring and alerts for connection failures

## Communication Plan

### Internal Team

- Notify engineering team immediately
- Share incident report with severity assessment
- Schedule post-mortem after fix is deployed

### Users

- Acknowledge issue if users report missing messages
- Provide ETA for fix
- Notify when fix is deployed
- No proactive announcement needed (fix should be transparent)

---

**Issue Created**: 2025-10-07
**Last Updated**: 2025-10-07
**Date Resolved (Gateway)**: 2025-10-07
**Resolution Time**: Same day
**Assigned To**: Backend Team
**Fix Status**:
- ✅ **Gateway Side**: RESOLVED - whatsmeow library updated
- ⏳ **Chatwoot Side**: Optional improvements pending
**Solution Applied**: Update whatsmeow library to `v0.0.0-20251005083110-4fe97da162dc`
**Related Documentation**: `docs/issues/ISSUE-001-PROFILE-PICTURE-PANIC.md`
**Related Commits**: whatsmeow PR#950, commit "fix including privacy token in GetProfilePictureInfo"
**Fix Deployed**: Awaiting staging and production deployment

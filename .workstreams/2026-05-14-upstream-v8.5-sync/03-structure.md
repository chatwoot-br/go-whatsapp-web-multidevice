# S — Structure (~2 pages target)

QRSPI S stage. "Like a C header file: signatures and types, not implementation."

Produced from D. Constrains P to agreed interfaces — P cannot invent new signatures.

## File inventory

_(Every file touched, marked: NEW / MODIFIED / DELETED. Group by directory.)_

```
src/
  infrastructure/
    chatwoot/                  # NEW (from upstream)
      client.go                # NEW
      types.go                 # NEW
  pkg/utils/
    phone.go                   # NEW or MODIFIED — depends on Q3
    general.go                 # MODIFIED — three-way merge zone
  cmd/
    rest.go                    # MODIFIED — chatwoot wiring
  ...
```

## Public-API signatures (no bodies)

_(Function/type signatures only. The exact shape of every new function and every modified-signature function. Use Go syntax.)_

```go
// chatwoot/client.go
type Client struct { ... }
func NewClient(cfg Config) *Client
func (c *Client) ForwardMessage(ctx context.Context, msg WebhookPayload) error

// pkg/utils/phone.go (Q3-dependent shape)
func NormalizePhone(raw string) (string, error)
func ValidateAndNormalizeJID(raw string) (types.JID, error)  // if Q3=B
```

## Webhook contract surface

_(Every event type the gateway emits post-upgrade — including new ones from upstream. One line per event.)_

| Event | Payload shape (top-level keys) | Source |
|---|---|---|
| `message` | `device_id`, `chat`, `from`, `body`, ... | unchanged |
| `chat.presence` | `device_id`, `chat`, `state`, ... | NEW from upstream `c428afa` |
| ... | | |

## Configuration / env-var surface

_(All env vars: existing + added + removed. The whole envelope.)_

| Env var | Default | Notes |
|---|---|---|
| `WHATSAPP_WEBHOOK_SECRET` | `secret` | unchanged |
| `CHATWOOT_BASE_URL` | _(none)_ | NEW from upstream |
| ... | | |

---

**Rules:**

- Implementations forbidden. Bodies stay empty.
- If you can't write the signature, the design isn't done — go back to D.
- P will reference S; signature drift between S and P means S was incomplete.

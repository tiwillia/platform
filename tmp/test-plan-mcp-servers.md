# Test Plan: Per-Session MCP Server Configuration

## Prerequisites

- Cluster: `api.okd1.timslab`
- Namespace: `ambient-code`
- Backend API: `https://backend-route-ambient-code.apps.okd1.timslab`
- Auth: Bearer token from a valid service account or user
- CRD updated with `spec.mcpServers` field

```bash
# Set up variables used throughout the test plan
export API=https://backend-route-ambient-code.apps.okd1.timslab
export PROJECT=ambient-code
export TOKEN=$(oc whoami -t)
```

---

## Test 1: CRD Accepts `spec.mcpServers`

**Goal:** Verify the updated CRD schema accepts the new mcpServers field.

```bash
# Create a minimal CR directly via kubectl with mcpServers
cat <<'EOF' | oc apply -f -
apiVersion: vteam.ambient-code/v1alpha1
kind: AgenticSession
metadata:
  name: test-mcp-crd-validation
  namespace: ambient-code
spec:
  displayName: "CRD MCP Validation"
  timeout: 300
  llmSettings:
    model: "claude-sonnet-4-5"
    temperature: 0.7
    maxTokens: 4000
  mcpServers:
  - name: test-http-server
    type: http
    url: "https://example.com/mcp"
  - name: test-stdio-server
    type: stdio
    command: uvx
    args: ["my-mcp-package", "--flag"]
    env:
      MY_VAR: "value"
      MY_SECRET: "${MCP_TEST_STDIO_SERVER_API_KEY}"
status:
  phase: Pending
EOF
```

**Expected:** CR created successfully. Verify:

```bash
oc get agenticsession test-mcp-crd-validation -n ambient-code -o jsonpath='{.spec.mcpServers}' | python3 -m json.tool
```

Should show both servers with all fields preserved.

**Cleanup:**
```bash
oc delete agenticsession test-mcp-crd-validation -n ambient-code
```

---

## Test 2: Create Session via API with MCP Servers

**Goal:** Verify the backend API accepts `mcpServers` in `CreateAgenticSessionRequest` and stores them on the CR.

### 2a: HTTP-type MCP server

```bash
curl -sk -X POST "$API/projects/$PROJECT/agentic-sessions" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "displayName": "MCP HTTP Test",
    "timeout": 300,
    "llmSettings": {"model": "claude-sonnet-4-5", "temperature": 0.7, "maxTokens": 4000},
    "mcpServers": [
      {
        "name": "context7",
        "type": "http",
        "url": "https://mcp.context7.com/mcp"
      }
    ]
  }' | python3 -m json.tool
```

**Expected:** 200 OK. Response includes the session with `spec.mcpServers` containing the server.

```bash
# Capture session name from response, e.g.:
export SESSION_HTTP=<session-name-from-response>

# Verify CR has mcpServers
oc get agenticsession $SESSION_HTTP -n $PROJECT -o jsonpath='{.spec.mcpServers}' | python3 -m json.tool
```

### 2b: Stdio-type MCP server with env placeholders

```bash
curl -sk -X POST "$API/projects/$PROJECT/agentic-sessions" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "displayName": "MCP Stdio Test",
    "timeout": 300,
    "llmSettings": {"model": "claude-sonnet-4-5", "temperature": 0.7, "maxTokens": 4000},
    "mcpServers": [
      {
        "name": "my-custom-tool",
        "type": "stdio",
        "command": "uvx",
        "args": ["my-custom-mcp", "--verbose"],
        "env": {
          "API_KEY": "${MCP_MY_CUSTOM_TOOL_API_KEY}",
          "BASE_URL": "https://api.example.com"
        }
      }
    ]
  }' | python3 -m json.tool
```

**Expected:** 200 OK. CR spec preserves command, args, and env (including `${MCP_*}` placeholders).

```bash
export SESSION_STDIO=<session-name-from-response>
oc get agenticsession $SESSION_STDIO -n $PROJECT -o jsonpath='{.spec.mcpServers[0].env}' | python3 -m json.tool
```

### 2c: Multiple MCP servers

```bash
curl -sk -X POST "$API/projects/$PROJECT/agentic-sessions" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "displayName": "MCP Multi Test",
    "timeout": 300,
    "llmSettings": {"model": "claude-sonnet-4-5", "temperature": 0.7, "maxTokens": 4000},
    "mcpServers": [
      {"name": "server-a", "type": "http", "url": "https://a.example.com/mcp"},
      {"name": "server-b", "type": "http", "url": "https://b.example.com/mcp"},
      {"name": "server-c", "type": "stdio", "command": "npx", "args": ["-y", "some-mcp-server"]}
    ]
  }' | python3 -m json.tool
```

**Expected:** 200 OK. All 3 servers appear in `spec.mcpServers`.

```bash
export SESSION_MULTI=<session-name-from-response>
oc get agenticsession $SESSION_MULTI -n $PROJECT -o jsonpath='{.spec.mcpServers}' | python3 -m json.tool
```

### 2d: Empty name filtered out

```bash
curl -sk -X POST "$API/projects/$PROJECT/agentic-sessions" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "displayName": "MCP Empty Name Test",
    "timeout": 300,
    "llmSettings": {"model": "claude-sonnet-4-5", "temperature": 0.7, "maxTokens": 4000},
    "mcpServers": [
      {"name": "", "type": "http", "url": "https://ignored.example.com"},
      {"name": "valid-server", "type": "http", "url": "https://valid.example.com/mcp"}
    ]
  }' | python3 -m json.tool
```

**Expected:** Only `valid-server` appears in `spec.mcpServers`. The empty-name entry is silently dropped.

```bash
export SESSION_EMPTY=<session-name-from-response>
oc get agenticsession $SESSION_EMPTY -n $PROJECT -o jsonpath='{.spec.mcpServers}' | python3 -m json.tool
```

---

## Test 3: Create Session Without MCP Servers (Backward Compatibility)

**Goal:** Verify sessions without `mcpServers` still work as before.

```bash
curl -sk -X POST "$API/projects/$PROJECT/agentic-sessions" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "displayName": "No MCP Servers",
    "timeout": 300,
    "llmSettings": {"model": "claude-sonnet-4-5", "temperature": 0.7, "maxTokens": 4000}
  }' | python3 -m json.tool
```

**Expected:** 200 OK. Response has no `mcpServers` field (or it's null/empty). Session creates normally.

```bash
export SESSION_NONE=<session-name-from-response>
oc get agenticsession $SESSION_NONE -n $PROJECT -o jsonpath='{.spec.mcpServers}'
# Expected: empty output
```

---

## Test 4: GET Session Returns `mcpServers` in Parsed Spec

**Goal:** Verify `parseSpec()` correctly reads mcpServers from the CR and returns them in the API response.

```bash
# Re-read a session created with mcpServers in test 2a
curl -sk "$API/projects/$PROJECT/agentic-sessions/$SESSION_HTTP" \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool
```

**Expected:** The `spec.mcpServers` array is present in the response with type, name, and url fields intact.

---

## Test 5: List Sessions Returns `mcpServers`

**Goal:** Verify listing sessions also returns the mcpServers field.

```bash
curl -sk "$API/projects/$PROJECT/agentic-sessions?limit=5" \
  -H "Authorization: Bearer $TOKEN" | python3 -c "
import sys, json
data = json.load(sys.stdin)
for item in data.get('items', []):
    name = item.get('metadata', {}).get('name', '?')
    servers = item.get('spec', {}).get('mcpServers', [])
    if servers:
        print(f'{name}: {len(servers)} MCP server(s) -> {[s[\"name\"] for s in servers]}')
    else:
        print(f'{name}: no MCP servers')
"
```

**Expected:** Sessions created in tests 2a-2d show their MCP servers. Others show none.

---

## Test 6: Update Session MCP Servers (Stopped Session)

**Goal:** Verify the PUT endpoint accepts `mcpServers` updates on stopped/pending sessions.

```bash
# First, stop one of the test sessions (or use one that's already Pending/Stopped)
# Use SESSION_HTTP from test 2a (it should be in Pending or Stopped state)

# Add a new MCP server to the session
curl -sk -X PUT "$API/projects/$PROJECT/agentic-sessions/$SESSION_HTTP" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "mcpServers": [
      {"name": "context7", "type": "http", "url": "https://mcp.context7.com/mcp"},
      {"name": "new-server", "type": "http", "url": "https://new.example.com/mcp"}
    ]
  }' | python3 -m json.tool
```

**Expected:** 200 OK. Response shows both servers in `spec.mcpServers`.

```bash
oc get agenticsession $SESSION_HTTP -n $PROJECT -o jsonpath='{.spec.mcpServers}' | python3 -m json.tool
```

### 6b: Remove all MCP servers

```bash
curl -sk -X PUT "$API/projects/$PROJECT/agentic-sessions/$SESSION_HTTP" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "mcpServers": []
  }' | python3 -m json.tool
```

**Expected:** 200 OK. `spec.mcpServers` is removed from the CR entirely.

```bash
oc get agenticsession $SESSION_HTTP -n $PROJECT -o jsonpath='{.spec.mcpServers}'
# Expected: empty output
```

### 6c: Re-add MCP servers after removal

```bash
curl -sk -X PUT "$API/projects/$PROJECT/agentic-sessions/$SESSION_HTTP" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "mcpServers": [
      {"name": "restored-server", "type": "stdio", "command": "uvx", "args": ["my-tool"]}
    ]
  }' | python3 -m json.tool
```

**Expected:** 200 OK. `spec.mcpServers` is back with the new server.

---

## Test 7: Update Blocked on Running Session

**Goal:** Verify that updating mcpServers is rejected while a session is Running.

```bash
# Find a running session (or skip if none available)
RUNNING_SESSION=$(oc get agenticsessions -n $PROJECT -o json | python3 -c "
import sys, json
data = json.load(sys.stdin)
for item in data.get('items', []):
    phase = item.get('status', {}).get('phase', '')
    if phase == 'Running':
        print(item['metadata']['name'])
        break
")

if [ -n "$RUNNING_SESSION" ]; then
  curl -sk -X PUT "$API/projects/$PROJECT/agentic-sessions/$RUNNING_SESSION" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"mcpServers": [{"name": "blocked", "type": "http", "url": "https://blocked.com"}]}' \
    | python3 -m json.tool
  echo "Expected: 409 Conflict"
else
  echo "SKIP: No running session available for this test"
fi
```

**Expected:** 409 Conflict with message "Cannot modify session specification while the session is running".

---

## Test 8: Operator Injects `MCP_SERVERS_JSON` Env Var

**Goal:** Verify the operator injects the env var into runner pods.

```bash
# Use a session that transitions to Running (needs repos + GitHub configured)
# Or inspect any running session's pod:
RUNNING_POD=$(oc get pods -n $PROJECT -l app=ambient-runner --no-headers 2>/dev/null | head -1 | awk '{print $1}')

if [ -n "$RUNNING_POD" ]; then
  oc get pod $RUNNING_POD -n $PROJECT -o jsonpath='{.spec.containers[?(@.name=="runner")].env}' | python3 -c "
import sys, json
envs = json.load(sys.stdin)
for e in envs:
    if e['name'] == 'MCP_SERVERS_JSON':
        print('FOUND MCP_SERVERS_JSON:', e.get('value', '<empty>'))
        break
else:
    print('MCP_SERVERS_JSON not found (expected if session has no mcpServers)')
"
else
  echo "SKIP: No running pods. Create a session with mcpServers and repos to test."
fi
```

**Expected:** If the session has `spec.mcpServers`, the pod's runner container has `MCP_SERVERS_JSON` set to the JSON array. If no mcpServers, the env var is absent.

To create a session that will actually run (needs a repo configured):
```bash
curl -sk -X POST "$API/projects/$PROJECT/agentic-sessions" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "displayName": "MCP Operator Env Test",
    "initialPrompt": "Say hello and stop.",
    "timeout": 300,
    "llmSettings": {"model": "claude-sonnet-4-5", "temperature": 0.7, "maxTokens": 4000},
    "repos": [{"url": "https://github.com/ambient-code/session-config-reference"}],
    "mcpServers": [
      {"name": "env-test-server", "type": "http", "url": "https://test.example.com/mcp"}
    ]
  }' | python3 -m json.tool
```

Then wait for the pod to be created and inspect it:
```bash
export SESSION_ENV=<session-name>
# Wait for pod creation (~10-30s)
sleep 15
POD=$(oc get pods -n $PROJECT -l agenticsession=$SESSION_ENV --no-headers | awk '{print $1}')
oc get pod $POD -n $PROJECT -o jsonpath='{.spec.containers[?(@.name=="runner")].env[?(@.name=="MCP_SERVERS_JSON")].value}' | python3 -m json.tool
```

---

## Test 9: Round-Trip Fidelity

**Goal:** Verify mcpServers survive create -> read -> update -> read without data loss.

```bash
# Create with complex mcpServers
curl -sk -X POST "$API/projects/$PROJECT/agentic-sessions" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "displayName": "Round-Trip Test",
    "timeout": 300,
    "llmSettings": {"model": "claude-sonnet-4-5", "temperature": 0.7, "maxTokens": 4000},
    "mcpServers": [
      {
        "name": "full-featured",
        "type": "stdio",
        "command": "uvx",
        "args": ["my-tool", "--config", "/etc/config.yaml"],
        "env": {
          "API_KEY": "${MCP_FULL_FEATURED_API_KEY}",
          "ENDPOINT": "https://api.example.com",
          "DEBUG": "true"
        }
      }
    ]
  }' | python3 -m json.tool
```

```bash
export SESSION_RT=<session-name>

# Read back
SPEC=$(curl -sk "$API/projects/$PROJECT/agentic-sessions/$SESSION_RT" \
  -H "Authorization: Bearer $TOKEN" | python3 -c "
import sys, json
data = json.load(sys.stdin)
servers = data.get('spec', {}).get('mcpServers', [])
print(json.dumps(servers, indent=2))
")
echo "$SPEC"
```

**Expected:** All fields preserved: name, type, command, 3 args, 3 env entries (including the `${MCP_*}` placeholder).

---

## Test 10: MCP Servers Coexist with Workflow and Repos

**Goal:** Verify mcpServers don't interfere with existing workflow and repos fields.

```bash
curl -sk -X POST "$API/projects/$PROJECT/agentic-sessions" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "displayName": "Full Feature Test",
    "timeout": 300,
    "llmSettings": {"model": "claude-sonnet-4-5", "temperature": 0.7, "maxTokens": 4000},
    "repos": [{"url": "https://github.com/ambient-code/session-config-reference"}],
    "activeWorkflow": {
      "gitUrl": "https://github.com/ambient-code/workflows",
      "branch": "main",
      "path": "workflows/bugfix"
    },
    "mcpServers": [
      {"name": "extra-tool", "type": "http", "url": "https://extra.example.com/mcp"}
    ]
  }' | python3 -m json.tool
```

**Expected:** Response has all three: `spec.repos`, `spec.activeWorkflow`, and `spec.mcpServers` populated.

```bash
export SESSION_FULL=<session-name>
oc get agenticsession $SESSION_FULL -n $PROJECT -o json | python3 -c "
import sys, json
data = json.load(sys.stdin)
spec = data.get('spec', {})
print('repos:', 'PRESENT' if spec.get('repos') else 'MISSING')
print('activeWorkflow:', 'PRESENT' if spec.get('activeWorkflow') else 'MISSING')
print('mcpServers:', 'PRESENT' if spec.get('mcpServers') else 'MISSING')
"
```

---

## Cleanup

```bash
# Delete all test sessions
for s in $SESSION_HTTP $SESSION_STDIO $SESSION_MULTI $SESSION_EMPTY $SESSION_NONE $SESSION_ENV $SESSION_RT $SESSION_FULL; do
  [ -n "$s" ] && oc delete agenticsession "$s" -n $PROJECT --ignore-not-found 2>/dev/null
done
echo "Cleanup complete"
```

---

## Test Summary

| # | Test | Validates |
|---|------|-----------|
| 1 | CRD validation | CRD schema accepts mcpServers array |
| 2a | Create with HTTP server | Backend serializes HTTP-type server to CR |
| 2b | Create with stdio server + env | Backend preserves command, args, env placeholders |
| 2c | Create with multiple servers | Multiple servers stored correctly |
| 2d | Empty name filtering | Invalid entries silently dropped |
| 3 | Create without mcpServers | Backward compatibility — existing sessions unaffected |
| 4 | GET returns mcpServers | parseSpec() correctly deserializes mcpServers |
| 5 | List returns mcpServers | Listing sessions includes mcpServers in spec |
| 6a | Update adds servers | PUT endpoint updates mcpServers on stopped session |
| 6b | Update removes all servers | Empty array clears mcpServers from CR |
| 6c | Update re-adds servers | Servers can be re-added after removal |
| 7 | Update blocked while running | PUT rejects changes on Running sessions |
| 8 | Operator env var injection | Runner pod gets MCP_SERVERS_JSON env var |
| 9 | Round-trip fidelity | All field types survive create -> read cycle |
| 10 | Coexistence with workflow/repos | mcpServers don't interfere with other spec fields |

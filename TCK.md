# ConfigHub + Kubernetes Mini TCK

**Technology Compatibility Kit for ConfigHub + Kubernetes Integration**

## Purpose

This minimal test verifies that ConfigHub and Kubernetes are working together correctly. It's designed to be:

- ✅ **Simple** - No dependencies on any specific project
- ✅ **Fast** - Completes in < 2 minutes
- ✅ **Thorough** - Tests the complete ConfigHub → Kubernetes flow
- ✅ **Self-cleaning** - All resources automatically cleaned up

## What It Tests

1. **ConfigHub API connectivity** - Can authenticate and create spaces/units
2. **Kubernetes cluster access** - Can create Kind cluster and deploy pods
3. **Worker installation** - Can install and connect ConfigHub worker
4. **Apply workflow** - Can apply ConfigHub units to Kubernetes
5. **Live state verification** - ConfigHub can read Kubernetes state

## Usage

### Direct Execution

```bash
curl -fsSL https://raw.githubusercontent.com/monadic/devops-sdk/main/test-confighub-k8s | bash
```

### From SDK Repository

```bash
cd /path/to/devops-sdk
./test-confighub-k8s
```

### From Project Repositories

Both `traderx` and `microtraderx` include wrapper scripts:

```bash
# In traderx or microtraderx
./test-confighub-k8s
```

## Prerequisites

- `cub` - ConfigHub CLI ([install](https://docs.confighub.com/cli/installation/))
- `kind` - Kubernetes in Docker ([install](https://kind.sigs.k8s.io/docs/user/quick-start/))
- `kubectl` - Kubernetes CLI ([install](https://kubernetes.io/docs/tasks/tools/))
- ConfigHub authentication: `cub auth login`

## What It Creates

The test creates minimal resources:

| Resource | Name | Purpose |
|----------|------|---------|
| Kind cluster | `confighub-tck` | Kubernetes test environment |
| ConfigHub space | `confighub-tck` | Configuration namespace |
| ConfigHub unit | `test-pod` | Nginx pod configuration |
| Worker | `tck-worker` | ConfigHub → Kubernetes bridge |
| Kubernetes pod | `test-pod` | Running nginx container |

**All resources are automatically deleted on exit** (success or failure).

## Expected Output

```
🧪 ConfigHub + Kubernetes Mini TCK
===================================

This test verifies:
  - ConfigHub API connectivity
  - Kubernetes cluster access
  - Worker installation
  - Unit apply workflow
  - Live state verification

Checking prerequisites...
✅ All prerequisites met

Step 1: Create Kind cluster
----------------------------
✅ Kind cluster created

Step 2: Create ConfigHub space
-------------------------------
✅ ConfigHub space created

Step 3: Create test unit (nginx pod)
-------------------------------------
✅ Unit created in ConfigHub

Step 4: Install ConfigHub worker
---------------------------------
✅ Worker installed and connected

Step 5: Apply unit to Kubernetes
---------------------------------
✅ Unit applied

Step 6: Verify deployment
-------------------------
Waiting for pod to be ready (max 60s)...
✅ Pod is ready in Kubernetes

Step 7: Verify ConfigHub live state
------------------------------------
✅ ConfigHub live state shows: Running

Step 8: Final verification
--------------------------
✅ Pod verified in Kubernetes: Running

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🎉 SUCCESS! ConfigHub + Kubernetes integration verified
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Summary:
  ✅ Kind cluster: confighub-tck
  ✅ ConfigHub space: confighub-tck
  ✅ ConfigHub unit: test-pod
  ✅ Worker: tck-worker (connected)
  ✅ Pod status: Running
  ✅ ConfigHub → Kubernetes flow: WORKING

Your ConfigHub + Kubernetes environment is correctly configured!

Note: All test resources will be cleaned up automatically.

🧹 Cleaning up test resources...
✅ Cleanup complete
```

## Troubleshooting

### Error: 'cub' command not found

```bash
brew install confighubai/tap/cub
```

### Error: ConfigHub authentication failed

```bash
cub auth login
```

### Error: 'kind' command not found

```bash
brew install kind
```

### Error: Pod did not become ready in time

Check Docker is running:
```bash
docker ps
```

Check Kind cluster health:
```bash
kubectl cluster-info --context kind-confighub-tck
```

## Integration with Projects

### MicroTraderX

Add to prerequisites section:
```markdown
### Pre-Flight Check

./test-confighub-k8s
```

### TraderX

Add to README:
```markdown
## Prerequisites

Before deploying, verify your environment:

./test-confighub-k8s
```

## Design Principles

1. **Zero project dependencies** - Works standalone, no imports from traderx/microtraderx
2. **Minimal resources** - 1 cluster, 1 space, 1 unit, 1 worker, 1 pod
3. **Fast execution** - Complete in < 2 minutes
4. **Clean exit** - Always cleanup, even on failure
5. **Clear output** - Step-by-step progress with emojis
6. **Verifiable** - Tests both ConfigHub API and Kubernetes state

## Exit Codes

- `0` - Success, all tests passed
- `1` - Failure, check output for details

## Use Cases

- **Pre-flight check** - Before starting tutorials
- **CI/CD validation** - Verify test environment
- **Debugging** - Isolate ConfigHub vs Kubernetes issues
- **Documentation** - Verify examples work
- **Onboarding** - New user environment validation

## Related Documentation

- [ConfigHub Documentation](https://docs.confighub.com)
- [Kind Documentation](https://kind.sigs.k8s.io)
- [MicroTraderX Tutorial](https://github.com/monadic/microtraderx)
- [TraderX Production Example](https://github.com/monadic/traderx)

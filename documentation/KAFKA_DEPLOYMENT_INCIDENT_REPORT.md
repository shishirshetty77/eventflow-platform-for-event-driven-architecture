# Kafka Deployment Incident Report

**Date:** 2025-11-30
**Component:** Kafka (Infrastructure)
**Environment:** EC2 (Kind Cluster)

## Incident Overview

The Kafka StatefulSet failed to initialize on the EC2 Kind cluster. The pod transitioned between `ImagePullBackOff` and `CrashLoopBackOff` states, preventing dependent services (Auth, Payments, Notification) from starting.

## Root Cause Analysis

### 1. Environment Variable Conflict (Critical)

- **Issue:** Kubernetes automatically injects environment variables for services. For the `kafka` service, it injected `KAFKA_PORT` (e.g., `tcp://10.96.x.x:9092`).
- **Impact:** The Confluent Kafka image's startup script (`/etc/confluent/docker/configure`) checks for the presence of `KAFKA_PORT`. If found, it assumes the user is attempting to use the deprecated `port` configuration. Since the Kubernetes-injected value is a URL and not a port number, the script failed with the error: `port is deprecated. Please use KAFKA_ADVERTISED_LISTENERS instead.`
- **Detection:** Identified by manually executing the configure script inside the pod with verbose logging (`bash -x`).

### 2. Incorrect Volume Mount Path

- **Issue:** The `statefulset.yaml` mounted the Persistent Volume Claim (PVC) to `/bitnami/kafka`.
- **Impact:** The `confluentinc/cp-kafka` image stores data in `/var/lib/kafka/data`. Because the volume was mounted elsewhere, Kafka wrote data to the container's ephemeral root filesystem. This filled up the container's overlay storage, leading to crashes and potential data loss upon pod restart.

### 3. Zookeeper Connection Timeout

- **Issue:** The default Zookeeper connection timeout was insufficient for the resource-constrained EC2 `micro` instance.
- **Impact:** Kafka timed out while attempting to connect to Zookeeper during startup, resulting in a `ZooKeeperClientTimeoutException`.

### 4. Disk Space Exhaustion

- **Issue:** The EC2 instance's root volume reached 95% usage.
- **Impact:** New images (specifically the large Confluent Kafka image) failed to pull, resulting in `ImagePullBackOff` errors.

## Resolution Steps

### 1. Fix Environment Variable Conflict

Updated the Kafka container's `command` in `k8s/base/infrastructure/kafka/statefulset.yaml` to explicitly unset `KAFKA_PORT` before running the configuration script.

```yaml
command:
  - /bin/sh
  - -c
  - |
    unset KAFKA_PORT
    echo "Starting configure..."
    /etc/confluent/docker/configure || { echo "Configure failed"; exit 1; }
    # ... rest of startup sequence
```

### 2. Correct Volume Mount

Updated the `volumeMounts` path in `k8s/base/infrastructure/kafka/statefulset.yaml` to match the Confluent image's data directory.

```yaml
volumeMounts:
  - name: data
    mountPath: /var/lib/kafka/data
```

### 3. Increase Timeout

Added the `KAFKA_ZOOKEEPER_CONNECTION_TIMEOUT_MS` environment variable to the StatefulSet configuration.

```yaml
env:
  - name: KAFKA_ZOOKEEPER_CONNECTION_TIMEOUT_MS
    value: "30000" # Increased to 30 seconds
```

### 4. Free Disk Space

Executed `docker system prune -af --volumes` on the EC2 instance to reclaim approximately 3GB of disk space, allowing image pulls to succeed.

## Verification

- **Kafka Status:** The `kafka-0` pod is now in the `Running` state.
- **Logs:** Startup logs confirm successful Zookeeper connection and broker initialization.
- **Dependent Services:** Services such as `auth`, `payments`, and `notification` have successfully connected to Kafka and transitioned to a healthy state.

# 📡 Atlas Tracker Service

**Status:** Functional Core Complete (Ingestion & Search)
**Date:** December 07, 2025

## ✅ Implemented Features

The functional core of the Tracker Service has been successfully implemented, covering both the high-throughput "Write Path" (Ingestion) and the primary "Read Path" (Geospatial Search).

### 1. Ingestion Pipeline (Write Path)
The service handles high-throughput driver location updates using an asynchronous event-driven architecture.
* **gRPC Handler:** Accepts `UpdateLocation` requests from client applications.
* **Kafka Producer:** Publishes events to the `driver-gps` topic, decoupling ingestion from storage processing.
* **Kafka Consumer:** A dedicated background worker consumes events from the topic.
* **Storage:** Persists location data to Redis using Geospatial commands (`GEOADD`).

### 2. Nearby Search (Read Path)
The service enables querying for available drivers within a specific radius, utilizing Redis's geospatial engine.
* **gRPC Handler:** `GetNearbyDrivers`.
* **Logic:** Executes Redis `GEORADIUS` commands to retrieve drivers sorted by proximity.
* **Optimization:** Utilizes `GeoRadius` over `GeoSearch` to ensure broad compatibility and stability across Redis client versions.

# 📡 Atlas Project - Learning Roadmap

## Upcoming Services & Technical Goals

### 1. Order Service (PostgreSQL)
* **Goal:** Manage Ride Lifecycle.
* **Tech:** PostgreSQL, GORM/SQLx.
* **Go Fundamental:** `context.Context` (Timeouts) and `select` for handling cancellation.

### 2. Gateway Service
* **Goal:** Aggregate data for frontend.
* **Tech:** HTTP/REST.
* **Go Fundamental:** `sync.WaitGroup`, **Fan-Out/Fan-In** pattern to query microservices in parallel.

### 3. History Service (MongoDB)
* **Goal:** Archive high-volume GPS logs.
* **Tech:** MongoDB (NoSQL).
* **Go Fundamental:** **Worker Pool** pattern, Buffered **Channels** to handle write pressure.

### 4. Wallet Service
* **Goal:** Handle money safely.
* **Tech:** Distributed Locking (Redis) or Local Locking.
* **Go Fundamental:** `sync.Mutex` to protect shared local state (Race Conditions).
---

## 🛠 Architecture Overview

```mermaid
sequenceDiagram
    participant Driver
    participant gRPC_Server
    participant Kafka
    participant Worker
    participant Redis

    Driver->>gRPC_Server: UpdateLocation(lat, long)
    gRPC_Server->>Kafka: Produce "driver-gps"
    gRPC_Server-->>Driver: 200 OK (Ack)
    
    loop Background Worker
        Worker->>Kafka: Fetch Message
        Kafka-->>Worker: LocationEvent
        Worker->>Redis: GEOADD atlas:tracker:positions
    end

    Note over Redis: Data is indexed and searchable via GeoRadius
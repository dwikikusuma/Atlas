# 📡 Atlas Tracker Service

**Status:** Functional Core Complete (Tracker & Order)
**Date:** December 12, 2025

## ✅ Implemented Features

The functional core of the Atlas system now includes both the Geospatial Tracker and the Order Management System.

### 1. Ingestion Pipeline (Tracker Service)
The service handles high-throughput driver location updates using an asynchronous event-driven architecture.
* **gRPC Handler:** Accepts `UpdateLocation` requests.
* **Kafka Producer:** Publishes events to the `driver-gps` topic.
* **Kafka Consumer:** Background worker persists data to Redis (`GEOADD`).

### 2. Nearby Search (Tracker Service)
Enables querying for available drivers within a specific radius.
* **Logic:** Executes Redis `GEORADIUS` commands to retrieve drivers sorted by proximity.

### 3. Order Management (Order Service)
Manages the end-to-end lifecycle of a ride, ensuring consistency and handling distributed driver matching.
* **Ride Lifecycle:** Manages the state machine: `CREATED` → `MATCHED` → `STARTED` → `FINISHED`.
* **Dispatch Integration:** Triggers asynchronous driver discovery via Kafka (`ride-dispatch`) and handles worker-based driver assignment.
* **Storage:** PostgreSQL (pgx/v5) with SQLC for type-safe database interactions.
* **Key Pattern:** Implements the **Transactional Outbox** concept (via Kafka) to decouple order creation from driver matching.

---

# 📡 Atlas Project - Learning Roadmap

## Upcoming Services & Technical Goals

### 1. Gateway Service (Next)
* **Goal:** Aggregate data for frontend.
* **Tech:** HTTP/REST.
* **Go Fundamental:** `sync.WaitGroup`, **Fan-Out/Fan-In** pattern to query microservices in parallel.

### 2. History Service (MongoDB)
* **Goal:** Archive high-volume GPS logs.
* **Tech:** MongoDB (NoSQL).
* **Go Fundamental:** **Worker Pool** pattern, Buffered **Channels** to handle write pressure.

### 3. Wallet Service
* **Goal:** Handle money safely.
* **Tech:** Distributed Locking (Redis) or Local Locking.
* **Go Fundamental:** `sync.Mutex` to protect shared local state (Race Conditions).

---

## 🛠 Architecture Overview

```mermaid
sequenceDiagram
    participant Client
    participant Order_Svc
    participant Kafka
    participant Dispatch_Svc
    participant Tracker_Svc

    Client->>Order_Svc: CreateOrder()
    Order_Svc->>Postgres: INSERT (Status: CREATED)
    
    Client->>Dispatch_Svc: RequestRide(OrderID)
    Dispatch_Svc->>Tracker_Svc: GetNearbyDrivers()
    Tracker_Svc-->>Dispatch_Svc: [DriverID]
    
    Dispatch_Svc->>Kafka: Publish "RideDispatched" (Key: OrderID)
    
    loop Order Worker
        Kafka->>Order_Svc: Consume Event
        Order_Svc->>Postgres: UPDATE (Status: MATCHED, DriverID)
    end
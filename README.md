# 🛡️ Fraud Core AI

Fraud Core AI is a high-performance, real-time transaction monitoring and analysis system designed to detect fraudulent financial activities. The project implements a full lifecycle of financial event processing: from anomaly detection to live threat visualization.

## 🚀 Tech Stack
* **Language:** Go 1.25
* **Message Broker:** Apache Kafka (Event-driven architecture, Pub/Sub)
* **Caching & Rate Limiting:** Redis (Velocity checks, TTL-based limits)
* **Storage:** PostgreSQL (Data persistence, connection pooling)
* **Communication:** gRPC (Low-latency inter-service communication)
* **Real-time UI:** WebSockets (Live streaming to the frontend)

## 🏗️ Architecture & Components
The system is built on modular principles and consists of three key components:

### 1. Simulator (Data Generator)
Streams transactions into Kafka, mimicking real-world user behavior. It generates both legitimate operations and suspicious patterns, such as anomalous amounts in foreign jurisdictions or sudden high-frequency crypto-exchange transactions.

### 2. Processor (The Core Engine)
The central decision-making engine built with **Enterprise-Grade** resilience:
* **Clean Architecture:** Strict separation of concerns. The `Usecase` layer dictates business rules, entirely decoupled from `Infrastructure` (DB/Kafka) via interfaces.
* **Highload Ready:** Implements robust PostgreSQL Connection Pooling (`MaxOpenConns`, `MaxIdleConns`) and Kafka batch reading to survive traffic spikes.
* **Velocity Checks:** Performs high-speed rate limiting via Redis (`INCR` + `EXPIRE`).
* **Hybrid Analysis:** Orchestrates gRPC requests to the AI Risk Engine, combining LLM verdicts with local heuristic rules.
* **Fail-safe Mechanism:** Automatically switches to "Fail-Safe / Velocity Only" mode if the AI service becomes unavailable, ensuring zero downtime.
* **Graceful Resource Management:** A custom `closer` package ensures safe teardown of DB, Redis, and Kafka connections to prevent memory leaks during shutdowns.

### 3. Dashboard (Real-time UI)
Implemented as an event aggregator:
* **WebSocket Hub:** Manages a pool of active connections using `sync.Mutex` to prevent race conditions.
* **Event Streaming:** Consumes AI verdicts from Kafka and broadcasts them to the frontend via WebSockets, rendering neon-styled threat alerts without page refreshes.

## 🛠️ Detection Logic & Heuristics
The system utilizes a multi-layered risk filter:
1. **Velocity Blocking:** Blocks users executing an abnormal number of transactions within a short timeframe, overriding AI if necessary.
2. **Heuristic Blocking:** Blocks transactions that significantly exceed a user's historical maximum (e.g., > 2x MaxTx for amounts > $500).
3. **AI Verdicts:** Evaluates geographical mismatches (e.g., transactions from Nigeria for local Ukrainian users) and merchant risks (P2P/Crypto).
4. **Confidence Scoring:** If the AI is uncertain (Confidence Score < 75%) but the amount is massive (>$10,000), the transaction is not blocked but flagged as `[PENDING REVIEW]` for manual anti-fraud officer inspection.


---

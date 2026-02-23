# Fraud Core AI

Fraud Core AI is a high-performance, real-time transaction monitoring and analysis system designed to detect fraudulent financial activities. The project implements a full lifecycle of financial event processing: from anomaly detection to live threat visualization.

## 🚀 Tech Stack
- **Language:** Go 1.25
- **Message Broker:** Apache Kafka (Event-driven architecture)
- **Caching & Statistics:** Redis (Velocity checks, fast limits)
- **Storage:** PostgreSQL (Data persistence and event logging)
- **Communication:** gRPC (Inter-service communication with AI Risk Engine)
- **Real-time UI:** WebSockets (Live streaming to the dashboard)

## 🏗️ Architecture & Components
The system is built on modular principles and consists of three key components:

1. **Simulator**  
   Streams transactions into Kafka, mimicking real-world user behavior. It generates both legitimate operations and suspicious patterns, such as anomalous amounts in foreign jurisdictions or crypto-exchange transactions.

2. **Processor**  
   The central decision-making engine that:
   - **Infrastructure Sync:** Automatically verifies and initializes required Kafka topics (raw-transactions, fraud-alerts) on startup.
   - **Velocity Checks:** Performs limit checks via Redis within every 60-second rolling window.
   - **Hybrid Analysis:** Orchestrates gRPC requests to the AI Risk Engine, combining AI verdicts with local heuristic rules.
   - **Cold Start Logic:** Implements adaptive blocking for new users, preventing false positives when historical data is unavailable.
   - **Fail-safe Mechanism:** Automatically switches to "Monitoring Only" mode if the AI service is unavailable.

3. **Dashboard**  
   Implemented as an event aggregator that:
   - **WebSocket Hub:** Manages a pool of active connections and broadcasts alerts in real-time.
   - **Event Streaming:** Consumes analysis results from Kafka and broadcasts them to the frontend via WebSockets without requiring a page refresh.

## 🛠️ Detection Logic
The system utilizes a multi-layered risk filter:
- **Heuristic Blocking:** Blocks transactions that significantly exceed a user's historical maximum (for amounts > 500 USD).
- **AI Verdicts:** Evaluates geographical mismatches (e.g., transactions from Nigeria or Singapore for local users) and merchant risks (P2P/Crypto).
- **Confidence Scoring:** If the AI is uncertain (Confidence Score < 75%), the transaction is not blocked but is instead flagged as [PENDING REVIEW].

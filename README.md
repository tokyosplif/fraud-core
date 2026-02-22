# Fraud Detection System (Core)

A high-performance, real-time transaction monitoring and analysis system designed to detect fraudulent financial activities. The project implements a full lifecycle of financial event processing: from anomaly detection to live threat visualization.

## Tech Stack
- **Language:** Go 1.25
- **Message Broker:** Apache Kafka (Event-driven architecture)
- **Caching & Rate Limiting:** Redis (Velocity checks)
- **Storage:** PostgreSQL (Data persistence)
- **Communication:** gRPC (Inter-service communication)
- **Real-time UI:** WebSockets (Live Dashboard)

## Architecture & Components
1. **Simulator:** Streams transactions into Kafka, mimicking real-world user behavior and potential fraud patterns.
2. **Processor:** The decision-making engine that:
   - Performs **Velocity** limit checks via Redis (60-second rolling window).
   - Orchestrates gRPC requests to the AI Risk Engine for deep analysis.
   - Implements a **Fail-safe mechanism**: automatically switches to "Monitoring Only" mode if the AI service is unavailable.
3. **Dashboard:** An event aggregator that broadcasts analysis results to the frontend via WebSockets in real-time.


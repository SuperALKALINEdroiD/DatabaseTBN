# Database

This is a distributed, in-memory key-value store with write-ahead logging (WAL) and disk persistence.

## Features
- **Distributed KV Store**: Each node acts as an independent key-value store.
- **Write-Ahead Logging (WAL) Replay**: Ensures data consistency by replaying logs at startup.
- **In-Memory Storage with Threshold**: Uses a configurable memory threshold before flushing to disk.
- **Multi-Node Architecture**: Supports multiple nodes with configurable settings.
- **Cluster Metadata Management**: Stores and manages cluster state via `MetaDataConfig`.
- **Configurable Storage Modes**: Supports `KV` and `Logs`  (in development) store modes.

## Work in Progress
- [ ] **Old Config Loading** and validation: Currently, a new config is created each time.
- [ ] **WAL Replay**: Implementing full WAL replay logic to restore previous state.
- [ ] **Persistent Disk Storage**: Working on storing data to disk efficiently.
- [ ] **Additional Endpoints**: Implementing `GET`, `UPDATE`, `DELETE`, and other operations.
- [ ] **GraphQL & Live Queries**: Exploring GraphQL or similar solutions for reactivity.
- [ ] Implement red black tree by myself

## Future Plans
- **Replication & Failover**: Implement strategies for high availability.
- **Optimized Storage Management**: Improve memory-to-disk transition mechanisms.
- **Advanced Querying**: Support complex queries beyond simple KV operations.
- **Monitoring & Observability**: Add logging, metrics, and dashboard support.

Stay tuned for updates! 🚀


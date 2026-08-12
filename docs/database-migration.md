# 🗄️ Agro-Shield Database Migration

## Overview

Agro-Shield initially used SQLite during the MVP development phase. SQLite allowed the team to quickly prototype the application, test database operations, and develop the initial farmer registration and authentication features with minimal infrastructure.

As the project moves toward real-world deployment, the database architecture is being migrated from SQLite to PostgreSQL.

---

## Why We Are Migrating

PostgreSQL provides a stronger foundation for Agro-Shield as the platform grows.

The migration is intended to provide:

- Better support for concurrent users
- Strong transactional capabilities
- Improved data integrity
- Better scalability
- Production and cloud deployment readiness
- Support for larger farmer and buyer datasets
- A stronger foundation for marketplace transactions
- Better support for future analytics and reporting

---

## Database Evolution

### Initial MVP

```text
Go Application
      │
      ▼
Repository Layer
      │
      ▼
SQLite
```

### Current Architecture

```text
Handlers
    ↓
Services
    ↓
Repositories
    ↓
Database Layer
    ↓
SQLite → PostgreSQL
```

### Target Production Architecture

Handlers
    ↓
Services
    ↓
Repositories
    ↓
PostgreSQL
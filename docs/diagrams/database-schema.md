# Database Schema

```text
+------------------------------------------------+
|                  FARMERS                        |
+------------------------------------------------+
| id (INTEGER PRIMARY KEY)                        |
| full_name(first and last name)                                       |
| phone (UNIQUE)                                  |
| password_hash                                   |
| location                                        |
| created_at                                      |
| updated_at                                      |
+------------------------------------------------+
```

## Future Tables

```text
FARMERS
    │
    ├───────────┐
    │           │
    ▼           ▼
PRODUCT      MARKETPLACE
    │           │
    ▼           ▼
STORAGE      ORDERS
                │
                ▼
            PAYMENTS
```

Future additions include:

- Products
- Marketplace Listings
- Orders
- Reviews
- Notifications
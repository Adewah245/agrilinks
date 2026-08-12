# System Architecture

```text
                    +---------------------+
                    |      Web Browser    |
                    +----------+----------+
                               |
                               | HTTP Request
                               |
                    +----------v----------+
                    |       Router        |
                    +----------+----------+
                               |
                    +----------v----------+
                    |     Middleware      |
                    | Validation/Filters  |
                    +----------+----------+
                               |
                    +----------v----------+
                    |      Handlers       |
                    +----------+----------+
                               |
                    +----------v----------+
                    |      Services       |
                    | Business Logic      |
                    +----------+----------+
                               |
                    +----------v----------+
                    |    Repository       |
                    | SQL Operations      |
                    +----------+----------+
                               |
                    +----------v----------+
                    | SQLite Database     |
                    +---------------------+
```

### Description

- Browser sends requests.
- Router directs requests.
- Middleware validates requests.
- Handlers receive user input.
- Services process business logic.
- Repository communicates with SQLite.
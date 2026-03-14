---
name: prod-sql
description: Query the production PocketBase SQLite database via SSH for debugging. Use when the user says "query prod", "check prod db", "prod sql", "debug prod", "what's in prod", or wants to inspect production data.
---

# Prod SQL

Run read-only SQL queries against the production PocketBase database for debugging.

## Connection

```
ssh root@46.225.161.186 "sqlite3 /var/lib/private/rekan-prod/data.db \"<QUERY>\""
```

## Rules

- SELECT only. Never run INSERT, UPDATE, DELETE, DROP, or ALTER.
- Use `substr()` to truncate large text columns in output.
- Start by listing tables (`.tables`) or checking schema (`.schema <table>`) if unsure about structure.
- Show the query you're about to run before running it.

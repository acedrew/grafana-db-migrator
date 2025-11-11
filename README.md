# Grafana SQLite to Postgres Database Migrator
[![Go Report Card](https://goreportcard.com/badge/github.com/percona/grafana-db-migrator)](https://goreportcard.com/report/github.com/percona/grafana-db-migrator)

## Background

This project was forked from [wbh1/grafana-sqlite-to-postgres](https://github.com/wbh1/grafana-sqlite-to-postgres) and enhanced with numerous fixes and improvements for Grafana 9.2.20 compatibility.

## Key Features & Improvements (2025-11)

### Enhanced Data Integrity
- **Preserves LogQL query backticks**: Fixes JSON parsing errors in alert rules by preserving backtick string delimiters used in LogQL queries
- **Boolean column verification**: Automatically detects and fixes boolean columns that weren't properly converted from SQLite integers, preventing authentication errors
- **Comprehensive hex data conversion**: Converts hex-encoded data for dashboards, data sources, alert rules, alert configurations, and more
- **Schema compatibility fixes**: Handles schema mismatches between SQLite and PostgreSQL schemas (e.g., `is_paused` column)

### Operational Improvements
- **Auto-cleanup of default data**: Removes auto-created organizations and data sources that Grafana generates during initial schema setup
- **Detailed progress logging**: Real-time progress tracking with statement-by-step counters
- **Multi-architecture support**: Docker builds support both amd64 and arm64 (Apple Silicon)
- **Database reset automation**: Included `reset-db.sh` script for repeatable testing and deployment

### Migration Process Enhancements
- **Duplicate key handling**: Gracefully skips duplicate entries without failing the migration
- **Type conversion fixes**: Handles PostgreSQL BYTEA type conversion issues automatically
- **Improved regex sanitization**: More precise SQL dump sanitization that preserves important data
- **Empty statement handling**: Properly handles trailing empty statements in large dumps

## Prerequisites
You **must** already have an existing database in Postgres for Grafana.

Run `CREATE DATABASE grafana` in `psql` to make the database. Then, start up an instance of Grafana pointed to the new database. Grafana will automagically create all the tables that it will need. You can shut Grafana down once those tables are made. We **need** those tables to exist for the migration to work.

**Important**: Use the included `reset-db.sh` script to automate this process. The script:
1. Stops Grafana
2. Drops and recreates the database schema
3. Starts Grafana to initialize tables
4. Waits for initialization to complete
5. Stops Grafana so migration can proceed

## Compatibility
Tested on:

| OS             | SQLite Version | Postgres Version | Grafana Version | Last Tested |
| -------------- | -------------- | ---------------- | --------------- | ----------- |
| MacOS          | 3.24.0         | 11.3             | 6.1.0+          | 2019        |
| CentOS 7/RHEL7 | 3.7.17         | 11.3             | 6.1.0+          | 2019        |
| Fedora 36      | 3.36.0         | 15.0             | 9.2.0           | 2022        |
| Ubuntu 22.04   | 3.37.2         | 14.x             | 9.2.20          | 2025-11     |

## Usage
```
usage: Grafana SQLite to Postgres Migrator [<flags>] <sqlite-file> <postgres-connection-string>

A command-line application to migrate Grafana data from SQLite to Postgres.

Flags:
  --help                      Show context-sensitive help (also try --help-long and --help-man).
  --dump=/tmp                 Directory path where the sqlite dump should be stored.
  --debug                     Enable debug level logging
  --reset-home-dashboard      Reset home dashboard for default organization
  --change-char-to-text       Change CHAR field to TEXT
  --fix-folders-id            Fix correlation between folders and dashboards

Args:
  <sqlite-file>                 Path to SQLite file being imported.
  <postgres-connection-string>  URL-format database connection string to use in the URL format (postgres://USERNAME:PASSWORD@HOST/DATABASE).
```

### Use as Docker image
1. Build docker image (multi-arch support):
   ```bash
   docker build -t grafana-db-migrator .
   ```

2. Run migration:
   ```bash
   docker run --rm -ti --network=host \
     -v <PATH_TO_DB_FILE>:/grafana.db \
     grafana-db-migrator:latest \
     --debug /grafana.db \
     "postgres://<USERNAME>:<PASSWORD>@<HOST>:5432/<DATABASE_NAME>?sslmode=disable"
   ```

### Complete Migration Workflow

1. **Prepare the PostgreSQL database**:
   ```bash
   # Ensure Grafana is stopped and database is fresh
   ./reset-db.sh
   ```

2. **Run the migration**:
   ```bash
   docker run --rm -ti --network=host \
     -v ./grafana.db:/grafana.db \
     grafana-db-migrator:latest \
     --debug /grafana.db \
     "postgres://grafana:grafana@localhost:5432/grafana?sslmode=disable"
   ```

3. **Verify the migration**:
   - Check the output for `✅ All boolean columns are correctly typed`
   - Check for `📊 Import summary` showing successful statement counts
   - Verify no fatal errors occurred

4. **Start Grafana with PostgreSQL**:
   ```bash
   # Update grafana.ini or environment variables to use PostgreSQL
   # Then start Grafana
   sudo systemctl start grafana-server
   ```

## Example Command
```bash
./grafana-migrate --debug grafana.db "postgres://postgres:PASSWORD@localhost:5432/grafana?sslmode=disable"
```

Notice the `?sslmode=disable` parameter. The [pq](https://github.com/lib/pq) driver has sslmode turned on by default, so you may need to add a parameter to adjust it. You can see all the support connection string parameters [here](https://godoc.org/github.com/lib/pq#hdr-Connection_String_Parameters).

## How it works
1. **Dump**: Exports SQLite database to SQL format in /tmp directory
2. **Sanitize**: Transforms SQLite-specific syntax to PostgreSQL-compatible SQL:
   - Removes SQLite-specific statements (PRAGMA, BEGIN, sqlite_sequence)
   - Preserves backticks in LogQL queries while converting table name backticks to quotes
   - Removes schema-incompatible columns (e.g., `is_paused` in alert_rule)
   - Converts hex-encoded data (`X'...'` → `'\x...'`)
   - Removes migration_log and alert_configuration_history entries
3. **Prepare**: Converts boolean columns to integers for import compatibility
4. **Import**: Executes sanitized SQL statements against PostgreSQL
   - Handles duplicate key errors gracefully
   - Converts hex data to UTF-8 text in JSON columns
5. **Post-process**:
   - Converts integer columns back to boolean
   - Verifies all boolean conversions succeeded
   - Fixes any remaining integer boolean columns
   - Fixes sequence values for auto-increment columns
6. **Cleanup**: Removes auto-generated default organizations and data sources

## Troubleshooting

### Authentication Errors After Migration
**Error**: `operator does not exist: integer = boolean`

**Cause**: Boolean columns weren't properly converted from SQLite integers.

**Solution**: The migration tool automatically detects and fixes this. Check logs for:
```
✅ All boolean columns are correctly typed
```
or
```
⚠️  Found X columns that are still integers, converting to boolean...
```

If you still see authentication errors, manually verify:
```sql
SELECT column_name, data_type
FROM information_schema.columns
WHERE table_name = 'user'
AND column_name IN ('is_admin', 'email_verified', 'is_disabled', 'is_service_account');
```

All should show `boolean`, not `integer`.

### Alert Rules Failing to Load
**Error**: `invalid character 'b' after object key:value pair` or similar JSON parsing errors

**Cause**: LogQL queries use backticks as string delimiters, which were being incorrectly converted to escaped quotes.

**Solution**: This is fixed in the current version. The sanitization process now preserves backticks within JSON data while only converting backticks around table names.

### Dashboard HTTP 500 Errors
**Cause**: Hex-encoded dashboard data wasn't converted to UTF-8 text.

**Solution**: Ensure `dashboard.data` is in the `HexDataChanges` array (already fixed in current version). The migration converts hex data to text for all dashboard, alert, and configuration columns.

### Empty Data Sources with Configuration Errors
**Error**: `Failed to load the data source configuration for Prometheus: Failed creating data source proxy`

**Cause**: Grafana auto-creates default data sources during schema initialization.

**Solution**: The migration tool automatically deletes these auto-created data sources before importing your real configurations.

### Large Database Performance
For databases with millions of statements (4M+):
- Migration typically takes 2-3 hours
- Use `--debug` flag to see detailed progress
- Monitor progress with: `Progress: X/Y statements processed`
- Ensure adequate disk space in `/tmp` for the SQL dump (typically 2-3x the SQLite file size)

## Known Limitations

- **Import Speed**: Currently processes statements individually rather than in batches for reliability. Large databases (4M+ statements) will take several hours to migrate.
- **Schema Version Sensitivity**: Tested specifically with Grafana 9.2.20. Other versions may have schema differences requiring adjustments.
- **Memory Usage**: Large SQL dumps are loaded into memory for processing. Ensure adequate RAM (recommend 8GB+ for very large databases).

## Technical Details

### Files Modified From Original Fork
- `pkg/postgresql/grafana.go`: Added hex conversion for multiple tables, boolean verification function
- `pkg/sqlite/sanitize.go`: Modified to preserve backticks in JSON data, added schema compatibility fixes
- `cmd/grafana-migrate/main.go`: Added boolean verification step, updated logging
- `pkg/postgresql/import.go`: Enhanced duplicate handling, added progress logging
- `Dockerfile`: Added multi-architecture support
- `Makefile`: Updated for native architecture builds
- `reset-db.sh`: New automation script for database preparation

### Data Integrity Guarantees
- All dashboards, panels, and visualizations are preserved
- Alert rules and notification channels are maintained
- User accounts, teams, and permissions are migrated
- Folder hierarchies and organization structure are retained
- Data source configurations are transferred (excluding auto-generated defaults)
- API keys and auth tokens are preserved

## Contributing

Contributions are welcome! This tool has been significantly enhanced for Grafana 9.2.20 compatibility. If you encounter issues with other Grafana versions or have improvements, please submit issues or pull requests.

## Acknowledgments
Inspiration for this program was taken from
- [wbh1/grafana-sqlite-to-postgres](https://github.com/wbh1/grafana-sqlite-to-postgres)
- [haron/grafana-migrator](https://github.com/haron/grafana-migrator)
- [This blog post](https://0x63.me/migrating-grafana-from-sqlite-to-postgresql/)

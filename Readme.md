# Shopping List Server
This application include the server for handling changes from multiple users for a single shopping list.

```bash
./your-server-binary [options]
```

## Configuration Options
The following options to configure the application are available.

| Flag          | Type     | Default                    | Description                                       |
| ------------- | -------- | -------------------------- | ------------------------------------------------- |
| `-a`          | `string` | `0.0.0.0`                  | Server listen address.                            |
| `-p`          | `string` | `46152`                    | Server listen port.                               |
| `-cert`       | `string` | `resources/servercert.pem` | Path to the TLS certificate file.                 |
| `-key`        | `string` | `resources/serverkey.pem`  | Path to the TLS key file.                         |
| `-k`          | `bool`   | `false`                    | Disable TLS (for testing purposes).               |
| `-jwt`        | `string` | `resources/jwtSecret.json` | Path to the JWT secret file.                      |
| `-c`          | `string` | `resources/db.json`        | Path to the database configuration file.          |
| `-h`          | `string` | `localhost`                | Database host (used if `DB_HOST` env is not set). |
| `-reset`      | `bool`   | `false`                    | Reset the entire database on startup.             |
| `-l`          | `string` | `server.log`               | Path to the log file.                             |
| `-production` | `bool`   | `false`                    | Enable production mode.                           |

## Environment Variables

These environment variables can override parts of the database configuration:

| Variable      | Description                           |
| ------------- | ------------------------------------- |
| `DB_HOST`     | Hostname of the database server.      |
| `DB_PASSWORD` | Password for the database user.       |
| `DB_USER`     | Username for the database connection. |
| `DB_NAME`     | Name of the database to connect to.   |

## Example
```bash
DB_PASSWORD=supersecret DB_USER=admin ./your-server-binary -p 8080 -k -reset
```
# Notes

`Notes` is a Go web server for serving private notes or posts behind token-based authorization.

## Core Functionality

- **Storage Engine**: Reads content from the local filesystem or synchronizes dynamically from a remote Git repository.
- **Access Control**: Configurable access tokens. Using magic links, grants users time-locked access to specific content.

## API

### Public Data

- **`GET /~/*`**: Serves static web assets directly from the static content directory without authentication middleware.

### Administration Endpoints

- **`GET,POST /~/admin/link`**: Renders the token generation form (accesses with basic authentication). Accepts form parameters and returns the generated magic login link containing the encrypted token.

### Protected Data Routes

- **`GET /*`**: Serves protected assets and documents. Requires a valid token to access.

## Token Configuration

Access permissions and token behavior are configured through the internal magic link generator endpoint. Tokens use AES-GCM encryption to encode parameters into a single signed string stored as encrypted cookies.

###  Token Parameters

The token structure supports the following configuration fields:

| Parameter | Type         | Description                                                                                           |
| :-------- | :----------- | :---------------------------------------------------------------------------------------------------- |
| `nbf`     | `*time.Time` | Format: `2006-01-02T15:04`. Triggers the countdown timer view if the request occurs before this time. |
| `naf`     | `*time.Time` | Format: `2006-01-02T15:04`. Restricts logins to times before this threshold.                          |
| `exp`     | `*time.Time` | Format: `2006-01-02T15:04`. Absolute expiration time; the token becomes invalid after this point.     |
| `home`    | `string`     | Default landing page or root path assigned to the token session.                                      |
| `allows`  | `[]string`   | Whitelist of allowed path prefixes accessible under this token.                                       |

## Configuration

The application is configured using the following environment variables:

| Variable        | Type   | Default | Description                                                   |
| :-------------- | :----- | :------ | :------------------------------------------------------------ |
| `LOG_FORMAT`    | string | `text`  | Output format for application logs (`json`, `text`).          |
| `USERNAME`      | string | `admin` | Set the basic auth username for admin endpoints.              |
| `PASSWORD`      | string | `admin` | Set the basic auth password for admin endpoints.              |
| `STORE`         | string | `fs`    | Storage backend driver (`fs` or `git`).                       |
| `STORE_FS_PATH` | string | *None*  | Path to the directory containing application data.            |
| `GIT_URL`       | string | *None*  | Remote repository target URL. Required when `STORE=git`.      |
| `GIT_USER`      | string | `git`   | Username for Git HTTP Basic Authentication.                   |
| `GIT_TOKEN`     | string | *None*  | Personal Access Token or password. Required when `STORE=git`. |

## Custom Countdowns

When a client accesses a path with a valid token where `nbf > current_time`, the request is halted by the countdown middleware.

### Template Resolution Order

1. **Path Override**: Checks for a custom layout located at `/<home>/countdown.html` within the active data directory. If present, it renders this file.
2. **Static Override**: Checks for a custom layout located at `/~/countdown.html` within the active data directory. If present, it renders this file.
3. **Built-in Fallback**: If no path override exists, it renders the default HTML view. See: [`countdown.html.tmpl`](internal/countdown/countdown.html.tmpl).

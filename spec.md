Go Secure Blog Server (SecureFileServe) Specification - Standard Library Only

1. Overview and Goals

The goal of this project is to create a high-performance, secure file server in Go that serves static blog content from the local file system. This application is constrained to use only the Go standard library for all functionality, ensuring a lightweight and dependency-free binary.

Core Features

High-Performance File Serving: Serve content from the file system, utilizing in-memory caching and standard HTTP caching headers.

Stateless Magic Link Authentication: Access permissions are fully contained within a versioned, encrypted, and Base64URL-encoded payload within the magic link URL.

Versioned Bearer Token: The encrypted payload includes a version identifier (V) for graceful future updates.

Time-Bound Access Control: Supports optional Not Before (NBF) and Expiration (EXP) timestamps. Access before NBF redirects to a countdown page.

Path-Based Authorization: Access is strictly controlled using glob patterns (/path/*, /file.html) contained in the token payload, verified against the requested URL path.

Admin Interface: A standard library HTTP handler for generating the encrypted magic link URLs.

2. High-Level Architecture (Standard Library Focus)

The application will use the net/http package for routing and serving, and the crypto packages for security.

Component

Responsibility

Standard Library Package

HTTP Router

Defines and routes all endpoints.

net/http

Auth Middleware

Token validation, time checks, and path glob matching.

net/http, crypto/aes, encoding/json, path

Static File Server

Serves content from the public /static directory.

net/http.FileServer

Restricted File Server

Serves content from the protected content root.

net/http.FileServer, io/fs

Token Service

Encrypts and decrypts access payloads using AES-GCM.

crypto/aes, crypto/cipher, encoding/base64, encoding/json

Caching

In-memory storage for file content.

sync (e.g., sync.Map)

3. Core Component Specification

3.1. Magic Link / Bearer Token Data Structure (Encrypted Payload)

This structure is used for both the magic link payload and the final cookie token.

3.2. Token Service (service/token)

The service must manage a single, high-entropy, static encryption key (loaded from an environment variable).

    Encryption: Use AES-256 GCM for authenticated encryption.

        Serialize the AccessPayload struct to JSON (encoding/json).

        Encrypt the JSON using AES-GCM.

        Base64URL-encode the result (ciphertext + nonce/IV).

    Decryption: Reverse the process, using the standard library's integrity check built into GCM to validate the token's authenticity.

3.3. File System Structure

    Restricted Content (/content):

        This is the root directory for content requiring the AUTH_TOKEN.

        A content directory typically contains one primary HTML file and its referenced static assets.

    Global Static Content (/static):

        Always Public. Used for the main login page, the NBF countdown page, and general site assets (logos, universal CSS).

4. Authentication Flow (Magic Link Redemption)

URL Format: GET /auth/login?token=<B64_ENCRYPTED_PAYLOAD>

    Handler Execution: The AuthHandler receives the request.

    Payload Extraction & Decryption:

        Extract the token query parameter.

        Decrypt the payload using the Token Service.

        Version Check: If decryption fails, or if AccessPayload.V is unknown (e.g., a version newer than the server supports), redirect to /unauthorized.

    Timestamp Validation:

        Expiration (EXP) Check: If time.Now() > AccessPayload.EXP, redirect to /expired.

        Not Before (NBF) Check: If time.Now() < AccessPayload.NBF:

            Redirect the user to the Countdown Page: /static/countdown.html?until=<timestamp> (where timestamp is NBF).

    Cookie Setting: If validation passes:

        Set a secure, HTTP-Only cookie (AUTH_TOKEN) with the original encrypted payload as its value.

        The cookie's expiration should be set to the token's EXP (if defined), ensuring browser cookies don't outlive the token.

    Final Redirect: Redirect the user to the first path listed in AccessPayload.Paths (e.g., /content/welcome/index.html).

5. Middleware and Access Control

5.1. Auth Middleware (middleware/auth)

This middleware will wrap the RestrictedFileServer handler.

    Cookie Extraction: Retrieve the AUTH_TOKEN cookie. If missing, redirect to /static/login.html.

    Token Validation: Decrypt the cookie value into an AccessPayload. If invalid or expired, clear the cookie and redirect to /static/login.html.

    Authorization (Glob Matching):

        Use the Standard Library's path.Match(pattern, name string) function for glob matching.

        The requested path (r.URL.Path) must be normalized and checked against every pattern in AccessPayload.Paths.

        If no pattern matches the requested path, return 403 Forbidden.

    Success: Call the next handler (RestrictedFileServer).

5.2. Content Serving and Caching

    Caching: A simple in-memory sync.Map will be used to cache the byte content of frequently requested files to avoid redundant disk I/O. Cache entries should have a TTL (e.g., 5 minutes) or be monitored for file changes if using a more advanced approach (still within standard library, e.g., polling file modification times).

    File Serving: Use net/http.FileServer but customize its serving logic to prepend the content root (/content) and apply appropriate HTTP caching headers.

5.3. Routing Table

All routing uses the standard library's net/http.ServeMux.
Route	Middleware	Handler	Note
/static/	None	http.FileServer(http.Dir("./static"))	Public access for static assets.
/auth/login	None	AuthHandler	Handles magic link redemption.
/auth/token	None	AdminHandler	Page/API for generating links.
/	AuthMiddleware	RestrictedFileServer	Protected content root.

6. Admin Interface Specification (Link Generation)

The Admin interface is a simple web form or handler that generates the encrypted magic link string.

    Input: The administrator provides:

        List of Allowed Paths (globs).

        Optional NBF datetime.

        Optional EXP datetime.

    Generation Logic:

        Construct the AccessPayload (setting V=1).

        Serialize to JSON.

        Encrypt and Base64URL-encode via the Token Service.

    Output: Display the final, ready-to-share URL: /auth/login?token=<ENCRYPTED_PAYLOAD>.

7. Implementation Environment

All implementation must strictly adhere to using only the Go standard library packages. Key packages include: net/http, encoding/json, encoding/base64, crypto/aes, crypto/cipher, time, path, and sync.
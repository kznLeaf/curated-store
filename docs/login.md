# Authentication and Login Technical Documentation

This document outlines the authentication architecture and login flow for the **Curated Store** project, focusing on the integration of **OAuth2 Proxy** and **Google OAuth 2.0**.

---

## Architecture Overview

The project implements a **Reverse Proxy Authentication** pattern. Instead of the application handling OAuth2 handshakes directly, a dedicated sidecar or gateway service (**OAuth2 Proxy**) sits in front of the `frontend` service.

* **OAuth2 Proxy**: Acts as the gatekeeper. It handles provider redirection, callback validation, and session cookie management.
* **Frontend Service**: Operates behind the proxy. It trusts the identity headers passed by the proxy and manages application-level session persistence.

```mermaid

```

---

## Authentication Flow

### A. Middleware Pipeline
Every request to the frontend passes through a chain of Go middlewares configured in `main.go`:

1.  **`otelhttp`**: OpenTelemetry instrumentation for tracing.
2.  **`authorize`**: Validates user identity from headers and enforces whitelist rules.
3.  **`ensureSessionID`**: Manages the `shop_session-id` cookie used for cart persistence.
4.  **`logHandler`**: Records structured logs including `session_id` and `request_id`.

### B. The Login Procedure
When a user clicks "Login" or accesses a protected resource, the following happens:

1.  **`loginHandler`**: 
    * Checks for an existing `Authorization` header. If present, redirects to home.
    * If not authenticated, redirects the user to `/oauth2/start?rd={return_to}`, which is intercepted by the OAuth2 Proxy.
2.  **Provider Handshake**: OAuth2 Proxy redirects the user to Google. Upon successful login, Google redirects back to the proxy's `/oauth2/callback`.
3.  **Header Injection**: Once authenticated, the proxy forwards the request to the Go frontend, injecting the following headers:
    * `X-Forwarded-User`: The unique user identifier.
    * `X-Forwarded-Email`: The user's email address.
    * `Authorization`: The Bearer token (if configured).

```mermaid
sequenceDiagram
autonumber
participant B as Browser
participant EXT as frontend-external(LB Service)
participant P as oauth2-proxy
participant F as frontend
participant G as Google OAuth2/OIDC

Note over B,P: 场景B：先访问 /login
B->>EXT: GET /login
EXT->>P: 转发请求
P->>F: /login 属于白名单，放行
F-->>B: 302 /oauth2/start?rd=/
B->>P: GET /oauth2/start?rd=/
P-->>B: 302 跳转 Google 授权端点
B->>G: 登录并授权
G-->>B: 重定向 /oauth2/callback
B->>P: GET /oauth2/callback
P->>G: code 换 token
G-->>P: token 与用户身份
P-->>B: 302 / + Set-Cookie(_oauth2_proxy)
B->>EXT: GET / + Cookie:_oauth2_proxy
EXT->>P: 转发请求
P->>F: 注入身份头后转发
F-->>B: 200 首页
```

![](img/loginA.svg)

---

## Whitelist Strategy (Public Routes)

To ensure high performance and accessibility for public content, specific paths bypass mandatory authentication.

* **Bypass Logic**: Defined in `IsAuthWhitelistPath`.
* **Whitelisted Paths**:
    * `/` (Home page)
    * `/_healthz` (Liveness probes)
    * `/login`
    * `/static/*` (CSS, Images, JS)
    * `/product/*` (Product details)
    * `/setCurrency`

These paths are configured both in the Go `authorize` middleware and the OAuth2 Proxy `OAUTH2_PROXY_SKIP_AUTH_ROUTES` environment variable to ensure consistency.

---

## Session & Identity Management

### User Context
The `authorize` middleware extracts identity from headers and injects them into the Go `context.Context` using `ctxKeyUserID` and `ctxKeyEmail`. Downstream handlers can retrieve these using the `userID(r)` helper function.

---

## 5. Security Configuration (K8s)

The `oauth2-proxy` is deployed with the following critical security settings in `oauth2-proxy.yaml`:

| Variable | Value | Description |
| :--- | :--- | :--- |
| `OAUTH2_PROXY_HTTP_ADDRESS` | `0.0.0.0:4180` | Internal proxy port. |
| `OAUTH2_PROXY_COOKIE_HTTPONLY` | `true` | Prevents JS access to the auth cookie. |
| `OAUTH2_PROXY_COOKIE_SAMESITE` | `lax` | Prevents CSRF while allowing cross-site navigation. |
| `OAUTH2_PROXY_PASS_AUTHORIZATION_HEADER` | `true` | Passes the ID Token to the frontend. |

> Note on Signature Verification: While `verifier.go` provides logic to verify Google's public keys via OIDC, this feature is currently disabled in `middleware.go` for environments with restricted access to Google endpoints 

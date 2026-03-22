# Authentication and Login Technical Documentation

This document outlines the authentication architecture and login flow for the **Curated Store** project, focusing on the integration of **OAuth2 Proxy** and **Google OAuth 2.0**.

---

## Architecture Overview

The project implements a **Reverse Proxy Authentication** pattern. Instead of the application handling OAuth2 handshakes directly, a dedicated sidecar or gateway service (**OAuth2 Proxy**) sits in front of the `frontend` service.

```mermaid
graph LR
    subgraph Public_Network [External]
        Browser[Web Browser]
        Google[Google OAuth 2.0]
    end

    subgraph Kubernetes_Cluster [Kubernetes]
        subgraph Proxy_Pod [Pod: oauth2-proxy]
            ProxyApp[oauth2-proxy Engine]
        end

        subgraph Frontend_Pod [Pod: frontend]
            Middleware[Middleware Stack]
            Handlers[Business Handlers]
        end
    end

    Browser -->|1. Request localhost:8080| ProxyApp
    ProxyApp <-->|2. Authenticate| Google
    ProxyApp -->|3. Set _oauth2_proxy Cookie| Browser
    
    ProxyApp ==>|4. Forward with X-Forwarded-User/Email| Middleware
    Middleware --> Handlers
```

* **OAuth2 Proxy**: Acts as the gatekeeper. It handles provider redirection, callback validation, and session cookie management.
* **Frontend Service**: Operates behind the proxy. It trusts the identity headers passed by the proxy and manages application-level session persistence.

---

## Authentication Flow

### The Login Procedure

When a user clicks "Login" or accesses a protected resource, the following happens:

1.  **`loginHandler`**: 
    * Checks for an existing `Authorization` header. If present, redirects to home.
    * If not authenticated, redirects the user to `/oauth2/start?rd={return_to}`, which is intercepted by the OAuth2 Proxy.
2.  **Provider Handshake**: OAuth2 Proxy redirects the user to Google. Upon successful login, Google redirects back to the proxy's `/oauth2/callback`.
3.  **Header Injection**: Once authenticated, the proxy forwards the request to the Go frontend, injecting the following headers:
    * `X-Forwarded-User`: The unique user identifier.
    * `X-Forwarded-Email`: The user's email address.
    * `Authorization`: The Bearer token (if configured).

### Whitelist Strategy (Public Routes)

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

The image below shows what happens when a user accesses a **protected** resource:

```mermaid
sequenceDiagram
autonumber
participant B as Browser
participant EXT as frontend-external(LB Service)
participant P as oauth2-proxy
participant F as frontend
participant G as Google OAuth2/OIDC

Note over B,P: 场景A：直接访问受保护页面（如 /cart）
B->>EXT: GET /cart
EXT->>P: 转发请求
P-->>B: 302 /oauth2/start?rd=/cart

B->>P: GET /oauth2/start?rd=/cart
P-->>B: 302 跳转 Google 授权端点
B->>G: 打开登录/同意页
G-->>B: 登录成功后重定向 /oauth2/callback?code&state
B->>P: GET /oauth2/callback?code&state
P->>G: 用 code 换 token 并校验 ID Token
G-->>P: token 与用户身份
P-->>B: 302 /cart + Set-Cookie(_oauth2_proxy)

B->>EXT: GET /cart + Cookie:_oauth2_proxy
EXT->>P: 转发请求
P->>P: 校验会话 Cookie
P->>F: 转发上游并注入 X-Forwarded-User/X-Forwarded-Email
F->>F: authorize 中间件写入 ctxKeyUserID/ctxKeyEmail
F-->>P: 200 页面
P-->>B: 200 页面
```

---

If the path is **whitelisted**:

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


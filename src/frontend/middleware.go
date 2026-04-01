// Copyright 2018 Google LLC
// Copyright 2026 kznLeaf
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// 中间件，通过装饰器模式实现了会话管理和日志记录的功能。
// 先执行 ensureSessionID 中间件，确保每个请求都有一个 sessionID，并将其注入到请求上下文中；
// 再执行 logHandler 中间件，记录请求的基本信息（路径、方法、请求 ID、会话 ID）以及请求完成时的状态码、响应大小和处理耗时等信息。

package main

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	auth "github.com/kznLeaf/curated-store/src/frontend/validator"
	"github.com/sirupsen/logrus"
)

type logHandler struct {
	log  *logrus.Logger
	next http.Handler
}

// responseRecorder 实现了http.ResponseWriter接口，用于构建HTTP响应
type responseRecorder struct {
	b      int
	status int
	w      http.ResponseWriter
}

func (r *responseRecorder) Header() http.Header { return r.w.Header() }

func (r *responseRecorder) Write(p []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.w.Write(p)
	r.b += n
	return n, err
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.w.WriteHeader(statusCode)
}

func (lh *logHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/_healthz") {
		lh.next.ServeHTTP(w, r)
		return
	}

	ctx := r.Context()
	requestID, _ := uuid.NewRandom()
	ctx = context.WithValue(ctx, ctxKeyRequestID{}, requestID.String())

	start := time.Now()
	rr := &responseRecorder{w: w}

	log := lh.log.WithFields(logrus.Fields{
		"http.req.path":   r.URL.Path,
		"http.req.method": r.Method,
		"http.req.id":     requestID.String(),
	})

	if v, ok := r.Context().Value(ctxKeySessionID{}).(string); ok {
		log = log.WithField("session", v)
	}

	if v, ok := r.Context().Value(ctxKeyUserID{}).(string); ok {
		log = log.WithField("userId", v)
	}

	defer func() {
		log.WithFields(logrus.Fields{
			"http.resp.took_ms": int64(time.Since(start) / time.Millisecond),
			"http.resp.status":  rr.status,
			"http.resp.bytes":   rr.b,
		}).Debugf("request complete")
	}()

	// 后续的 handler 可以通过 r.Context().Value(ctxKeyLog{}) 获取这个 logger
	ctx = context.WithValue(ctx, ctxKeyLog{}, log)
	r = r.WithContext(ctx)

	lh.next.ServeHTTP(rr, r)
}

// ensureSessionID 确保会话ID存在，并将会话 ID 保存到 context 中。
// 有下面几种情况：
//
// 1. 没有 cookie，但是环境变量指定了共享会话：使用同一个硬编码的会话ID
// 2. 没有 cookie，且环境变量指定了不共享会话：生成一个随机的 sessionID，不共享同一个会话
// 3. 没有 cookie，且发生的是其他类型的 err ：直接中断请求，不再往后执行
// 4. 有 cookie，且 cookie 无效：生成一个随机的 sessionID，不共享同一个会话
//
// 在引入 OAuth2 认证之前，用户身份信息是通过会话 ID 进行关联的，所以确保每个请求都有一个有效的 sessionID 是非常重要的。
// 现在 sessionID 不再和用户身份直接关联，但是仍予以保留
func ensureSessionID(next http.Handler) http.HandlerFunc {
	// Notice that "type HandlerFunc func(ResponseWriter, *Request)"
	// 也就是说实现了 "type Handler interface { ServeHTTP(ResponseWriter, *Request) }" 接口的函数会自动变成 HandlerFunc 类型
	// 所以只要返回一个 func(w http.ResponseWriter, r *http.Request) 签名的函数即可。
	return func(w http.ResponseWriter, r *http.Request) {
		var sessionID string
		c, err := r.Cookie(cookieSessionID)
		if err == http.ErrNoCookie {
			// 如果 Cookie 中不存在 sessionID，且环境变量正确，使用固定的 sessionID，所有用户共享同一个会话
			if os.Getenv("ENABLE_SINGLE_SHARED_SESSION") == "true" {
				// Hard coded user id, shared across sessions
				sessionID = "12345678-1234-1234-1234-123456789123"
			} else {
				// 没有 cookie 并且没有启用共享会话，说明是第一次请求，生成一个随机的 sessionID，不共享同一个会话
				u, _ := uuid.NewRandom()
				sessionID = u.String()
			}
			http.SetCookie(w, &http.Cookie{
				Name:   cookieSessionID,
				Value:  sessionID,
				MaxAge: cookieMaxAge,
			})
		} else if err != nil { // 发生其他错误，直接中断请求，不继续执行后续中间件
			return
		} else {
			sessionID = c.Value // Cookie 存在且有效，直接使用
		}
		// 将 sessionID 设置到请求上下文中，后续处理函数可以获取到
		ctx := context.WithValue(r.Context(), ctxKeySessionID{}, sessionID)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	}
}

// authorize handle authorization for incoming requests. Obtain userID and email from header, and store them in context for later use.
// If paths that are not in the whitelist, then will skip authorization and directly call the next handler.
func authorize(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if user, ok := ctx.Value(ctxKeyUserID{}).(string); !ok || strings.TrimSpace(user) == "" {
			if forwardedUser := strings.TrimSpace(r.Header.Get("X-Forwarded-User")); forwardedUser != "" {
				ctx = context.WithValue(ctx, ctxKeyUserID{}, forwardedUser)
			}
		}

		if email, ok := ctx.Value(ctxKeyEmail{}).(string); !ok || strings.TrimSpace(email) == "" {
			if forwardedEmail := strings.TrimSpace(r.Header.Get("X-Forwarded-Email")); forwardedEmail != "" {
				ctx = context.WithValue(ctx, ctxKeyEmail{}, forwardedEmail)
			}
		}

		r = r.WithContext(ctx)

		if auth.IsAuthWhitelistPath(r.URL.Path, baseUrl) {
			next.ServeHTTP(w, r)
			return
		}

		user, _ := ctx.Value(ctxKeyUserID{}).(string)
		email, _ := ctx.Value(ctxKeyEmail{}).(string)
		if strings.TrimSpace(user) == "" || strings.TrimSpace(email) == "" {
			http.Error(w, "Unauthorized: Missing User Identity", http.StatusUnauthorized)
			return
		}

		// 中国大陆环境下无法获取 Google 公钥，故注释掉签名验证的逻辑

		// authz := auth.ParseBearerToken(r.Header.Get("Authorization"))
		// if authz == "" {
		// 	log.Warn("请求头中缺失 Authorization Token 拒绝访问")
		// 	http.Error(w, "Unauthorized: Missing Token", http.StatusUnauthorized)
		// 	return
		// }

		// rawIDToken := auth.ParseBearerToken(authz)
		// log.Infof("rawIDToken: %s", rawIDToken)

		// verifier, err := auth.GetVerifier(ctx)
		// if err != nil {
		// 	log.Errorf("verifier initialization failed: %v", err)
		// 	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		// 	return
		// }

		// idToken, err := verifier.Verify(ctx, rawIDToken)
		// if err != nil {
		// 	log.Errorf("ID Token authorization failed: %v", err)
		// 	http.Error(w, "Unauthorized: Invalid Token", http.StatusUnauthorized)
		// 	return
		// }

		next.ServeHTTP(w, r)
	}
}

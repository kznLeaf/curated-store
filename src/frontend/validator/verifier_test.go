package validator

import (
	"context"
	"testing"
)

// go test -bench=. -benchmem

func BenchmarkVerifier_Verify(b *testing.B) {
	ctx := context.Background()
	// 确保单例已加载
	v, _ := GetVerifier(ctx)
	bearer := "Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6ImM0MWYxNDFhYTE5ZGYwYWM5N2RhYTU1ZTYwMDc2NmM0YzUzNjRjNDIiLCJ0eXAiOiJKV1QifQ.eyJpc3MiOiJhY2NvdW50cy5nb29nbGUuY29tIiwiYXpwIjoiNDgwMDQ2MzE0ODk2LWI4aHVsNzFtN2hzY3Q1djl0MGRnb2hwZ3FxbXM1aXIxLmFwcHMuZ29vZ2xldXNlcmNvbnRlbnQuY29tIiwiYXVkIjoiNDgwMDQ2MzE0ODk2LWI4aHVsNzFtN2hzY3Q1djl0MGRnb2hwZ3FxbXM1aXIxLmFwcHMuZ29vZ2xldXNlcmNvbnRlbnQuY29tIiwic3ViIjoiMTE0OTU2MTEyOTk0NTE3MzkxMzkzIiwiZW1haWwiOiJrdW5wZW5nZHU0M0BnbWFpbC5jb20iLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwiYXRfaGFzaCI6InlTMDJrdlBoeHYyUXZtQThtc3dza3ciLCJpYXQiOjE3NzQwNzc4NTMsImV4cCI6MTc3NDA4MTQ1M30.1TRtfYsAwGbHpz2JLPVS0y8EQa3R5NUz572IbAZVtx4spLbOKryB3Jinl3pP8a0gvrXK10YlM5MwI5YJQLcQhdodXGTX3WASJsW_Q_M9ohLorEIptrTMXGxdrjyL7uQhjLOz4YvtIDB-8Rf_x7JTL9MwfMsa-wy7yyKJJpX3c4ie4AWMzNArJSJV3A1HLU0A26bbCc8Fx5NeEfl-SOySlUKU8FCyMF0dRLVYXS8JvHmP6qHh2KJSNXxzZsxrxHLLYGJYNU_pyuA8_wRqDPAFnGsfleXRtGOnibVrXV3ISRVp7FQGxIZ3sUrkHtphM34yyr1OnuMadNNwLV1mnUtdAA"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rawToken := ParseBearerToken(bearer)
		_, err := v.Verify(ctx, rawToken)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// go test -v -run TestIsAuthWhitelistPath

func TestIsAuthWhitelistPath(t *testing.T) {
	baseUrl := ""

	tests := []struct {
		name        string
		requestPath string
		baseUrl     string
		want        bool
	}{
		{"Root path", "/", "", true},
		{"Login path", "/login", "", true},
		{"Health check", "/_healthz", "", true},
		{"Currency toggle", "/setCurrency", "", true},
		{"Product detail", "/product/OLJCESPC7Z", "", true},

		// --- 场景 2: 带有 baseUrl 的匹配 (模拟生产环境) ---
		{"BaseUrl: Root match", "/shop/", "/shop", true},
		{"BaseUrl: Login match", "/shop/login", "/shop", true},
		{"BaseUrl: Health check", "/shop/_healthz", "/shop", true},

		// --- 场景 3: 路由中存在但不在白名单中的路径 (需要鉴权的) ---
		{"Cart view (should be false)", "/cart", "", false},
		{"Checkout (should be false)", "/cart/checkout", "", false},
		{"empty (should be false)", "/cart/empty", "", false},
		{"Assistant (should be false)", "/assistant", "", false},

		// --- 场景 4: 静态资源路径 (关键：对应 PathPrefix) ---
		// 注意：如果你的函数目前只做 map 匹配，这里会返回 false。
		// 建议在函数中加入 strings.HasPrefix(requestPath, "/static/") 逻辑。
		{"Static Assets CSS", "/static/css/main.css", "", true},

		// --- 场景 5: 边界与异常情况 ---
		{"Trailing slash mismatch", "/login/", "", false}, // map 匹配通常不带斜杠
		{"Substring attack", "/login-fake", "", false},
		{"BaseUrl manipulation", "/shoplogin", "/shop", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseUrl = tt.baseUrl // 注入当前用例的 baseUrl
			if got := IsAuthWhitelistPath(tt.requestPath, baseUrl); got != tt.want {
				t.Errorf("IsAuthWhitelistPath(%q) with baseUrl %q = %v, want %v",
					tt.requestPath, tt.baseUrl, got, tt.want)
			}
		})
	}
}

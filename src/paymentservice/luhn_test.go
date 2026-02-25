package main

import "testing"

func TestLuhnCheck(t *testing.T) {
	tests := []struct {
		name   string
		number string
		want   bool
	}{
		// ── 合法卡号（真实测试卡号，Luhn 校验通过）──
		{"visa standard", "4111111111111111", true},
		{"visa 2", "4480600782503898", true},
		{"visa 13-digit", "4222222222222", true},
		{"mastercard", "5500005555555559", true},
		{"mastercard2", "5350757731695581", true},
		{"mastercard 2-series", "2221000000000009", true},
		{"amex", "378282246310005", true},
		{"discover", "6011111111111117", true},
		// 带连字符和空格，应能正确过滤
		{"with hyphens", "4111-1111-1111-1111", true},
		{"with spaces", "4111 1111 1111 1111", true},

		// ── 非法卡号（checksum 错误）──
		{"wrong checksum", "4111111111111112", false},
		{"all zeros 16-digit", "0000000000000000", false},
		{"random digits", "1234567890123456", false},

		// ── 边界情况 ──
		{"too short 12-digit", "411111111111", false}, // len < 13
		{"exactly 13-digit valid", "4222222222222", true},
		{"empty string", "", false},
		{"only spaces", "    ", false},
		{"non-numeric chars only", "abcd-efgh", false},
		{"mixed valid digits with letters", "4111abc1111111111", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := luhnCheck(tt.number)
			if got != tt.want {
				t.Errorf("luhnCheck(%q) = %v, want %v", tt.number, got, tt.want)
			}
		})
	}
}

func TestDetectCardType(t *testing.T) {
	tests := []struct {
		name     string
		number   string
		cardType string
	}{
		// ── Visa ──
		{"visa 16-digit", "4111111111111111", "visa"},
		{"visa 13-digit", "4222222222222", "visa"},
		{"visa my", "4480600782503898", "visa"},
		{"visa with hyphens", "4111-1111-1111-1111", "visa"},
		{"visa with spaces", "4111 1111 1111 1111", "visa"},

		// ── MasterCard 51-55 系列 ──
		{"mastercard 51", "5100005555555559", "mastercard"},
		{"mastercard 52", "5200005555555557", "mastercard"},
		{"mastercard 55", "5500005555555559", "mastercard"},
		{"mastercard my", "5350757731695581", "mastercard"},

		// ── MasterCard 2221-2720 系列 ──
		{"mastercard 2-series lower bound", "2221000000000009", "mastercard"},
		{"mastercard 2-series upper bound", "2720999999999996", "mastercard"},
		{"mastercard 2-series middle", "2500000000000001", "mastercard"},

		// ── 边界：刚好不在 MasterCard 2-series 范围内 ──
		{"just below 2221", "2220999999999999", "unknown"},
		{"just above 2720", "2721000000000000", "unknown"},
		// 51-55 边界
		{"just below 51", "5099999999999999", "unknown"},
		{"just above 55", "5600000000000000", "unknown"},

		// ── 其他卡类型 ──
		{"amex", "378282246310005", "unknown"},
		{"discover", "6011111111111117", "unknown"},
		{"diners", "30569309025904", "unknown"},
		{"jcb", "3530111333300000", "unknown"},

		// ── 边界情况 ──
		{"empty string", "", "unknown"},
		{"too short", "411111111111", "unknown"}, // 12位，< 13
		{"only hyphens", "----", "unknown"},
		{"only spaces", "    ", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectCardType(tt.number)
			if got != tt.cardType {
				t.Errorf("detectCardType(%q) = %q, want %q", tt.number, got, tt.cardType)
			}
		})
	}
}

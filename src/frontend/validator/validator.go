package validator

import "github.com/go-playground/validator/v10"

var validate *validator.Validate

func init() {
	validate = validator.New(validator.WithRequiredStructEnabled())
}

// SetCurrencyPayload 定义设置货币类型的 请求 的 数据结构。
//
// 字段说明：
//   - Currency: 货币代码，必填，需符合 ISO 4217 标准（如 USD、EUR、CNY）
//
// 验证规则：
//   - required: 字段不能为空
//   - iso4217: 必须是有效的 ISO 4217 货币代码（3 位字母）
type SetCurrencyPayload struct {
	Currency string `validate:"required,iso4217"`
}

func (sc *SetCurrencyPayload) Validate() error {
	return validate.Struct(sc)
}
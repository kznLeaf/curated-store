package validator

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
)

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

// PlaceOrderPayload 定义下单时提交的订单数据结构。
//
// 字段说明：
//   - Email: 用户邮箱，必填，需符合邮箱格式
//   - StreetAddress: 街道地址，必填，最大 512 字符
//   - ZipCode: 邮政编码，必填
//   - City: 城市，必填，最大 128 字符
//   - State: 州/省，必填，最大 128 字符
//   - Country: 国家，必填，最大 128 字符
//   - CcNumber: 信用卡号，必填，需符合信用卡格式（Luhn 算法验证）
//   - CcMonth: 信用卡过期月份，必填，范围 1-12
//   - CcYear: 信用卡过期年份，必填
//   - CcCVV: 信用卡 CVV 码，必填
//
// 验证规则：
//   - required: 字段不能为空
//   - email: 必须是有效的邮箱地址
//   - max=N: 字符串最大长度为 N
//   - credit_card: 必须是有效的信用卡号
//   - gte=1,lte=12: 月份范围 1-12
type PlaceOrderPayload struct {
	Email         string `validate:"required,email"`
	StreetAddress string `validate:"required,max=512"`
	ZipCode       int64  `validate:"required"`
	City          string `validate:"required,max=128"`
	State         string `validate:"required,max=128"`
	Country       string `validate:"required,max=128"`
	CcNumber      string `validate:"required,credit_card"`
	CcMonth       int64  `validate:"required,gte=1,lte=12"`
	CcYear        int64  `validate:"required"`
	CcCVV         int64  `validate:"required"`
}

// AddToCartPayload 定义添加商品到购物车的请求数据结构。
//
// 字段说明：
//   - Quantity: 商品数量，必填，范围 1-10
//   - ProductID: 产品唯一标识符，必填
//
// 验证规则：
//   - required: 字段不能为空
//   - gte=1: 数量大于等于 1
//   - lte=10: 数量小于等于 10（限制单次添加数量）
type AddToCartPayload struct {
	Quantity  uint64 `validate:"required,gte=1,lte=10"`
	ProductID string `validate:"required"`
}

// Validate 实现 Payload 接口，验证添加到购物车的请求数据。
//
// 返回：
//   - error: 如果验证失败返回 validator.ValidationErrors，成功返回 nil
//
// 使用示例：
//
//	payload := &AddToCartPayload{
//	    Quantity: 5,
//	    ProductID: "OLJCESPC7Z",
//	}
//	if err := payload.Validate(); err != nil {
//	    // 处理验证错误
//	}
func (ad *AddToCartPayload) Validate() error {
	return validate.Struct(ad)
}

// Validate 实现 Payload 接口，验证订单提交的数据。
//
// 返回：
//   - error: 如果验证失败返回 validator.ValidationErrors，成功返回 nil
//
// 验证项包括：
//   - 邮箱格式是否正确
//   - 地址信息是否完整且长度合理
//   - 信用卡号是否有效（使用 Luhn 算法）
//   - 过期日期是否在合理范围内
//
// 使用示例：
//
//	payload := &PlaceOrderPayload{
//	    Email: "user@example.com",
//	    StreetAddress: "1600 Amphitheatre Parkway",
//	    ZipCode: 94043,
//	    // ... 其他字段
//	}
//	if err := payload.Validate(); err != nil {
//	    // 处理验证错误
//	}
func (po *PlaceOrderPayload) Validate() error {
	return validate.Struct(po)
}

// ValidationErrorResponse 将验证错误转换为用户友好的错误消息。
//
// 参数：
//   - err: 验证过程中产生的错误
//
// 返回：
//   - error: 格式化后的错误消息，包含所有验证失败的字段及原因
//
// 功能：
//   - 检查错误是否为 validator.ValidationErrors 类型
//   - 遍历所有验证错误，提取字段名和验证规则
//   - 生成格式化的错误消息，便于返回给前端展示
//
// 错误消息格式：
//
//	Field 'Email' is invalid: email
//	Field 'ZipCode' is invalid: required
//
// 使用示例：
//
//	if err := payload.Validate(); err != nil {
//	    userErr := ValidationErrorResponse(err)
//	    http.Error(w, userErr.Error(), http.StatusUnprocessableEntity)
//	}
func ValidationErrorResponse(err error) error {
	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return errors.New("invalid validation error format")
	}
	var msg string
	for _, err := range validationErrs {
		msg += fmt.Sprintf("Field '%s' is invalid: %s\n", err.Field(), err.Tag())
	}
	return errors.New(msg)
}

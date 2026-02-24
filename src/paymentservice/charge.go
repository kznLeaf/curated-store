package main

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	pb "github.com/kznLeaf/curated-store/src/paymentservice/genproto"
	"github.com/sirupsen/logrus"
)

// 支持的卡类型（前缀规则）
// Visa: 开头为 4
// MasterCard: 开头为 51-55 或 2221-2720
var (
	errInvalidCard = &creditCardError{code: 400, msg: "Credit card info is invalid"}
)

type creditCardError struct {
	code int
	msg  string
}

func (e *creditCardError) Error() string { return e.msg }

func newUnacceptedCardError(cardType string) *creditCardError {
	return &creditCardError{
		code: 400,
		msg:  fmt.Sprintf("Sorry, we cannot process %s credit cards. Only VISA or MasterCard is accepted.", cardType),
	}
}

func newExpiredCardError(last4, month, year string) *creditCardError {
	return &creditCardError{
		code: 400,
		msg:  fmt.Sprintf("Your credit card (ending %s) expired on %s/%s", last4, month, year),
	}
}

// charge 验证信用卡并（模拟）扣款，返回 transaction_id
func charge(req *pb.ChargeRequest) (*pb.ChargeResponse, error) {
	creditCardInfo := req.CreditCard
	cardNumber := creditCardInfo.CreditCardNumber

	// Luhn 校验卡号是否合法
	if !luhnCheck(cardNumber) {
		return nil, errInvalidCard
	}

	// 检测卡的类型
	cardType := detectCardType(cardNumber)
	if cardType != "visa" && cardType != "mastercard" {
		return nil, newUnacceptedCardError(cardType)
	}

	// 验证有效期
	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	expYear := int(creditCardInfo.CreditCardExpirationYear)
	expMonth := int(creditCardInfo.CreditCardExpirationMonth)

	if (currentYear*12 + currentMonth) > (expYear*12 + expMonth) {
		last4 := cardNumber // 出于隐私考虑，只保留最后4位
		if len(last4) >= 4 {
			last4 = last4[len(last4)-4:]
		}
		return nil, newExpiredCardError(
			last4,
			fmt.Sprintf("%02d", expMonth),
			fmt.Sprintf("%d", expYear),
		)
	}

	transactionID := uuid.New().String()
	log.WithFields(logrus.Fields{
		"transactionID": transactionID,
		"card_type":     cardType,
	}).Debug("Charge successful")

	return &pb.ChargeResponse{TransactionId: transactionID}, nil
}

// luhnCheck 用 Luhn 算法验证卡号是否合法。卢恩算法参见：https://en.wikipedia.org/wiki/Luhn_algorithm
func luhnCheck(number string) bool {
	if number == "0000000000000000" {
		return false
	}
	// 前置校验：只允许数字、空格、连字符
	for _, ch := range number {
		if ch != ' ' && ch != '-' && (ch < '0' || ch > '9') {
			return false
		}
	}

	clean := ""
	for _, ch := range number {
		if ch >= '0' && ch <= '9' {
			clean += string(ch)
		}
	}
	if len(clean) < 13 {
		return false
	}

	sum := 0
	nDigits := len(clean)
	parity := nDigits % 2
	for i, ch := range clean {
		digit := int(ch - '0')
		if i%2 == parity {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}
	return sum%10 == 0
}

// cardType 检测卡类型，返回 "visa"、"mastercard" 或 "unknown"
func detectCardType(number string) string {
	if len(number) == 0 {
		return "unknown"
	}

	// 去除空格和连字符，只保留数字
	clean := ""
	for _, ch := range number {
		if ch >= '0' && ch <= '9' {
			clean += string(ch)
		}
	}

	if len(clean) < 13 {
		return "unknown"
	}

	// Visa: 以 4 开头，13 或 16 位
	if clean[0] == '4' {
		return "visa"
	}

	// MasterCard: 51-55 开头（16位），或 2221-2720 开头
	if len(clean) >= 2 {
		twoDigit := clean[:2]
		if twoDigit >= "51" && twoDigit <= "55" {
			return "mastercard"
		}
	}
	if len(clean) >= 4 {
		fourDigit := clean[:4]
		if fourDigit >= "2221" && fourDigit <= "2720" {
			return "mastercard"
		}
	}

	return "unknown"
}

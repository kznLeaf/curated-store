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

// 独立的货币转换模块，将金额拆分成整数部分和纳秒小数部分，避免浮点数精度问题，并提供基本的货币运算函数，如加法、乘法等。
// 原项目中没有在函数签名中使用指针，会报警告。这里修正了这一问题。

package money

import (
	"errors"

	pb "github.com/kznLeaf/curated-store/src/checkoutservice/genproto"
)

const (
	nanosMin = -999999999
	nanosMax = +999999999
	nanosMod = 1000000000
)

var (
	ErrInvalidValue        = errors.New("one of the specified money values is invalid")
	ErrMismatchingCurrency = errors.New("mismatching currency codes")
)

// IsValid checks if specified value has a valid units/nanos signs and ranges.
func IsValid(m *pb.Money) bool {
	if m == nil {
		return false
	}
	return signMatches(m) && validNanos(m.GetNanos())
}

func signMatches(m *pb.Money) bool {
	if m == nil {
		return false
	}
	return m.GetNanos() == 0 || m.GetUnits() == 0 || (m.GetNanos() < 0) == (m.GetUnits() < 0)
}

func validNanos(nanos int32) bool { return nanosMin <= nanos && nanos <= nanosMax }

// IsZero returns true if the specified money value is equal to zero.
func IsZero(m *pb.Money) bool {
	return m != nil && m.GetUnits() == 0 && m.GetNanos() == 0
}

// IsPositive returns true if the specified money value is valid and is
// positive.
func IsPositive(m *pb.Money) bool {
	return IsValid(m) && (m.GetUnits() > 0 || (m.GetUnits() == 0 && m.GetNanos() > 0))
}

// IsNegative returns true if the specified money value is valid and is
// negative.
func IsNegative(m *pb.Money) bool {
	return IsValid(m) && (m.GetUnits() < 0 || (m.GetUnits() == 0 && m.GetNanos() < 0))
}

// AreSameCurrency returns true if values l and r have a currency code and
// they are the same values.
func AreSameCurrency(l, r *pb.Money) bool {
	if l == nil || r == nil {
		return false
	}
	return l.GetCurrencyCode() == r.GetCurrencyCode() && l.GetCurrencyCode() != ""
}

// AreEquals returns true if values l and r are the equal, including the
// currency. This does not check validity of the provided values.
func AreEquals(l, r *pb.Money) bool {
	if l == nil || r == nil {
		return false
	}
	return l.GetCurrencyCode() == r.GetCurrencyCode() &&
		l.GetUnits() == r.GetUnits() && l.GetNanos() == r.GetNanos()
}

// Negate returns the same amount with the sign negated.
func Negate(m *pb.Money) *pb.Money {
	if m == nil {
		return nil
	}
	return &pb.Money{
		Units:        -m.GetUnits(),
		Nanos:        -m.GetNanos(),
		CurrencyCode: m.GetCurrencyCode()}
}

// Must panics if the given error is not nil. This can be used with other
// functions like: "m := Must(Sum(a,b))".
func Must(v *pb.Money, err error) *pb.Money {
	if err != nil {
		panic(err)
	}
	return v
}

// Sum adds two values. Returns an error if one of the values are invalid or
// currency codes are not matching (unless currency code is unspecified for
// both).
func Sum(l, r *pb.Money) (*pb.Money, error) {
	if !IsValid(l) || !IsValid(r) {
		return nil, ErrInvalidValue
	} else if l.GetCurrencyCode() != r.GetCurrencyCode() {
		return nil, ErrMismatchingCurrency
	}
	units := l.GetUnits() + r.GetUnits()
	nanos := l.GetNanos() + r.GetNanos()

	if (units == 0 && nanos == 0) || (units > 0 && nanos >= 0) || (units < 0 && nanos <= 0) {
		// same sign <units, nanos>
		units += int64(nanos / nanosMod)
		nanos = nanos % nanosMod
	} else {
		// different sign. nanos guaranteed to not to go over the limit
		if units > 0 {
			units--
			nanos += nanosMod
		} else {
			units++
			nanos -= nanosMod
		}
	}

	return &pb.Money{
		Units:        units,
		Nanos:        nanos,
		CurrencyCode: l.GetCurrencyCode()}, nil
}

// MultiplySlow is a slow multiplication operation done through adding the value
// to itself n-1 times.
func MultiplySlow(m *pb.Money, n uint32) *pb.Money {
	if m == nil {
		return nil
	}
	out := m
	for n > 1 {
		out = Must(Sum(out, m))
		n--
	}
	return out
}

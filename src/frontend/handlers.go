package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/mux"
	pb "github.com/kznLeaf/curated-store/src/frontend/genproto"
	"github.com/sirupsen/logrus"
)

var (
	templates = template.Must(template.New(""). // 创建一个空模版对象
			Funcs(template.FuncMap{           // 添加自定义函数，让模版可以调用go函数
			"renderMoney":        renderMoney,
			"renderCurrencyLogo": renderCurrencyLogo,
		}).ParseGlob("templates/*.html")) // 解析所有 templates 目录下的 html 文件
	plat             platformDetails
	isCymbalBrand    = strings.ToLower(os.Getenv("CYMBAL_BRANDING")) == "true"
	assistantEnabled = strings.ToLower(os.Getenv("ENABLE_ASSISTANT")) == "true"
	frontendMessage  = strings.TrimSpace(os.Getenv("FRONTEND_MESSAGE"))
)

type platformDetails struct {
	css      string
	provider string
}
type ctxKeyLog struct{}

// ctxKeySessionID 定义一个零内存占用的、强类型的键，value为 sessionID
type ctxKeySessionID struct{}
type ctxKeyRequestID struct{}

func (fe *frontendServer) productHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		renderHTTPError(log, r, w, errors.New("product id not specified"), http.StatusBadRequest)
		return
	}
	// 添加调试日志
	log.WithField("id", id).WithField("currency", currentCurrency(r)).Debug("[producthandler]调试信息:")

	product, err := fe.GetProduct(r.Context(), id)
	if err != nil {
		log.Infof("无法获取产品 %v", err)
		renderHTTPError(log, r, w, errors.New("could not retrieve product"), http.StatusInternalServerError)
		return
	}

	currencies, err := fe.getCurrencies(r.Context())
	if err != nil {
		log.Infof("无法获取货币列表 %v", err)
		renderHTTPError(log, r, w, errors.New("无法获取货币列表"), http.StatusInternalServerError)
		return
	}

	// product.GetPriceUsd() 为空？
	price, err := fe.convertCurrency(r.Context(), product.GetPriceUsd(), currentCurrency(r))
	if err != nil {
		log.Infof("转换货币错误: %v", err)
		renderHTTPError(log, r, w, errors.New("无法转换货币"), http.StatusInternalServerError)
		return
	}

	wrappedProduct := struct {
		Item  *pb.Product
		Price *pb.Money
	}{product, price}

	// 渲染 product.html 模板，把填充后的页面写入 r
	if err := templates.ExecuteTemplate(w, "product", injectCommonTemplateData(r, map[string]interface{}{
		"product":    wrappedProduct,
		"currencies": currencies,
	})); err != nil {
		log.Println(err)
	}

}

// 渲染相关的函数
func renderHTTPError(log logrus.FieldLogger, r *http.Request, w http.ResponseWriter, err error, code int) {
	log.WithField("error", err).Error("request error")
	errMsg := fmt.Sprintf("%+v", err)

	w.WriteHeader(code)

	if templateErr := templates.ExecuteTemplate(w, "error", injectCommonTemplateData(r, map[string]interface{}{
		"error":       errMsg,
		"status_code": code,
		"status":      http.StatusText(code),
	})); templateErr != nil {
		log.Println(templateErr)
	}
}

func renderCurrencyLogo(currencyCode string) string {
	logos := map[string]string{
		"USD": "$",
		"HKD": "HK$",
		"JPY": "¥",
		"CNY": "¥",
	}

	logo := "¥" //default
	if val, ok := logos[currencyCode]; ok {
		logo = val
	}
	return logo
}

func renderMoney(money *pb.Money) string {
	currencyLogo := renderCurrencyLogo(money.GetCurrencyCode())
	return fmt.Sprintf("%s%d.%02d", currencyLogo, money.GetUnits(), money.GetNanos()/10000000)
}

// injectCommonTemplateData 注入通用的模板数据到模版中。
// 先创建通用数据，再把页面专属数据 payload 合并到一起返回
func injectCommonTemplateData(r *http.Request, payload map[string]interface{}) map[string]interface{} {
	data := map[string]interface{}{
		"session_id":        sessionID(r),
		"request_id":        r.Context().Value(ctxKeyRequestID{}),
		"user_currency":     currentCurrency(r),
		"platform_css":      plat.css,
		"platform_name":     plat.provider,
		"is_cymbal_brand":   isCymbalBrand,
		"assistant_enabled": assistantEnabled,
		"deploymentDetails": deploymentDetailsMap,
		"frontendMessage":   frontendMessage,
		"currentYear":       time.Now().Year(),
		"baseUrl":           baseUrl,
	}

	for k, v := range payload {
		data[k] = v
	}

	return data
}

///////////////////////////////////////////////

func sessionID(r *http.Request) string {
	v := r.Context().Value(ctxKeySessionID{})
	if v == nil {
		return ""
	}
	return v.(string)
}

// currentCurrency 从请求的 Cookie 中提取用户的货币,如果没有设置则返回JPY
func currentCurrency(r *http.Request) string {
	c, _ := r.Cookie(cookieCurrency)
	if c != nil {
		return c.Value
	}
	return "JPY"
}

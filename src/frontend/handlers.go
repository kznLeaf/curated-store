package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	// "github.com/gorilla/mux"
	// "github.com/sirupsen/logrus"
)

type platformDetails struct {
	css      string
	provider string
}

var (
	frontendMessage  = strings.TrimSpace(os.Getenv("FRONTEND_MESSAGE"))
	isCymbalBrand    = strings.ToLower(os.Getenv("CYMBAL_BRANDING")) == "true"
	assistantEnabled = strings.ToLower(os.Getenv("ENABLE_ASSISTANT")) == "true"
	// templates        = template.Must(template.New(""). // 创建一个空模版对象
	// 			Funcs(template.FuncMap{           // 添加自定义函数，让模版可以调用go函数
	// 		"renderMoney":        renderMoney,
	// 		"renderCurrencyLogo": renderCurrencyLogo,
	// 	}).ParseGlob("templates/*.html")) // 解析所有 templates 目录下的 html 文件
	plat platformDetails
)

type ctxKeyLog struct{}
type ctxKeyRequestID struct{}

// currentCurrency 从 cookie 获取当前货币
func currentCurrency(r *http.Request) string {
	c, _ := r.Cookie(cookieCurrency)
	if c != nil {
		return c.Value
	}
	return defaultCurrency
}

func (fe *frontendServer) homeHandler(w http.ResponseWriter, r *http.Request) {
	// 1. 日志记录：从请求上下文获取日志记录器，如果发生错误，会把日志记录下来
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	log.WithField("currency", currentCurrency(r)).Info("home")

	fmt.Println("成功调用首页处理函数！")
}

func (fe *frontendServer) productHandler(w http.ResponseWriter, r *http.Request) {
	// log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	// id := mux.Vars(r)["id"]
	fmt.Println("成功调用产品处理函数！")
	products, err := fe.GetProducts(r.Context())
	if err != nil {
		http.Error(w, "无法获取产品列表", http.StatusInternalServerError)
		return
	}
	fmt.Printf("[debug]产品信息：%v\n", products)
}

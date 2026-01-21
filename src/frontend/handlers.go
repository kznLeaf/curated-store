package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
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

func (fe *frontendServer) productHandler(w http.ResponseWriter, r *http.Request) {
	// log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	// id := mux.Vars(r)["id"]
	fmt.Println("成功调用产品处理函数！")
}

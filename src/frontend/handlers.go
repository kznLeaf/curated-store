package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

type platformDetails struct {
	css      string
	provider string
}

type ctxKeyLog struct{}

// currentCurrency 从 cookie 获取当前货币
func currentCurrency(r *http.Request) string {
	c, _ := r.Cookie(cookieCurrency)
	if c != nil {
		return c.Value
	}
	return defaultCurrency
}

// func (fe *frontendServer) homeHandler(w http.ResponseWriter, r *http.Request) {
// 	// 1. 日志记录：从请求上下文获取日志记录器，如果发生错误，会把日志记录下来
// 	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
// 	log.WithField("currency", currentCurrency(r)).Info("home")
// }

func (fe *frontendServer) productHandler(w http.ResponseWriter, r *http.Request) {
	// log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	id := mux.Vars(r)["id"]
	log.Infof("成功调用产品处理函数！ID: %s", id)
	products, err := fe.GetProducts(r.Context())
	if err != nil {
		http.Error(w, "无法获取产品列表", http.StatusInternalServerError)
		return
	}
	log.Infof("[debug]产品信息：%v", products)

	product, err := fe.GetProduct(r.Context(), id)
	if err != nil {
		log.Infof("无法获取产品信息，ID: %s, 错误: %v", id, err)
		http.Error(w, "无法获取产品信息", http.StatusInternalServerError)
		return
	}
	log.Infof("[debug]单个产品信息：%v\n", product)

	products, err = fe.SearchProducts(r.Context(), "Sunglasses")
	if err != nil {
		log.Infof("无法搜索产品，查询: %s, 错误: %v", "Sunglasses", err)
		http.Error(w, "无法搜索产品", http.StatusInternalServerError)
		return
	}
	log.Infof("[debug]搜索Sunglasses结果 %v\n", products)

}

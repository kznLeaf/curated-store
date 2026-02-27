package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/mux"
	pb "github.com/kznLeaf/curated-store/src/frontend/genproto"
	validator "github.com/kznLeaf/curated-store/src/frontend/validator"
	"github.com/sirupsen/logrus"
)

var (
	templates = template.Must(template.New(""). // 创建一个空模版对象
			Funcs(template.FuncMap{           // 添加自定义函数，让模版可以调用go函数
			"renderMoney":        renderMoney,
			"renderCurrencyLogo": renderCurrencyLogo,
		}).ParseGlob("templates/*.html")) // 解析所有 templates 目录下的 html 文件
	plat             *platformDetails
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

func (fe *frontendServer) homeHandler(w http.ResponseWriter, r *http.Request) {
	log.WithField("[homehandler]currency", currentCurrency(r)).Info("home")

	currencies, err := fe.getCurrencies(r.Context())
	if err != nil {
		log.Infof("无法获取货币列表 %v", err)
		renderHTTPError(log, r, w, errors.New("could not retrieve currencies"), http.StatusInternalServerError)
		return
	}

	products, err := fe.GetProducts(r.Context())
	if err != nil {
		log.Infof("无法获取产品列表 %v", err)
		renderHTTPError(log, r, w, errors.New("could not retrieve products"), http.StatusInternalServerError)
		return
	}

	type productView struct {
		Item  *pb.Product // Product (string)类型定义在 proto，包含一个Picture字段，用于显示产品图片
		Price *pb.Money
	}
	ps := make([]productView, len(products))
	for i, p := range products {
		price, err := fe.convertCurrency(r.Context(), p.GetPriceUsd(), currentCurrency(r))
		if err != nil {
			log.Infof("转换货币错误: %v", err)
			renderHTTPError(log, r, w, errors.New("无法转换货币"), http.StatusInternalServerError)
			return
		}
		ps[i] = productView{p, price}
	}

	// 设置平台信息
	var env = os.Getenv("ENV_PLATFORM") // 如果没有该环境变量，说明是local环境。GCP会设置该环境变量.
	if env == "" || !stringinSlice(validEnvs, env) {
		log.Infof("当前环境不属于支持的云平台")
		env = "local"
	}

	addrs, err := net.LookupHost("metadata.google.internal.") // 查询域名的IP地址，只有内部的实例才能得到查询结果
	if err == nil && len(addrs) >= 0 {
		log.Debugf("Detected Google metadata server: %v, setting ENV_PLATFORM to GCP.", addrs)
		env = "gcp"
	}

	log.Debugf("当前环境: %s", env)
	plat = &platformDetails{}
	plat.setPlatformDetails(strings.ToLower(env))

	if err := templates.ExecuteTemplate(w, "home", injectCommonTemplateData(r, map[string]interface{}{
		"show_currency": true,
		"currencies":    currencies,
		"products":      ps,
		"banner_color":  os.Getenv("BANNER_COLOR"),                 // TODO  cart_size
		"ad":            nil, // home.html 里完全没有调用 {{ template "text_ad" }} 的代码，这里实际上是一个无效传入。
	})); err != nil {
		log.Error(err)
	}
}

var validEnvs = []string{"local", "gcp", "azure", "aws", "onprem", "alibaba"}

// stringinSlice 判断字符串 val 是否在字符串 slice 切片中
func stringinSlice(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// setPlatformDetails 用于渲染首页左侧的 platform-flag 元素，在 homehandler 完成赋值，后续plat被传入injectCommonTemplateData
func (plat *platformDetails) setPlatformDetails(env string) {
	switch env {
	case "aws":
		plat.provider = "AWS"
		plat.css = "aws-platform"
	case "onprem":
		plat.provider = "On-Premises"
		plat.css = "onprem-platform"
	case "azure":
		plat.provider = "Azure"
		plat.css = "azure-platform"
	case "gcp":
		plat.provider = "Google Cloud"
		plat.css = "gcp-platform"
	case "alibaba":
		plat.provider = "Alibaba Cloud"
		plat.css = "alibaba-platform"
	default:
		plat.provider = "local"
		plat.css = "local"
	}
}

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

	recommendations, err := fe.getRecommendations(r.Context(), sessionID(r), nil) // TODO 第三个参数为nil,也就是随机从所有产品中抽取
	if err != nil {
		log.Infof("获取推荐产品失败: %v", err)
	}

	wrappedProduct := struct {
		Item  *pb.Product
		Price *pb.Money
	}{product, price}

	// 渲染 product.html 模板，把填充后的页面写入 r
	if err := templates.ExecuteTemplate(w, "product", injectCommonTemplateData(r, map[string]interface{}{
		"ad":              fe.chooseAd(r.Context(), product.Categories, log),
		"show_currency": true,
		"product":       wrappedProduct,
		"currencies":    currencies,
		"recommendations":  recommendations,
		// TODO packagingInfo cart_size 待补充
	})); err != nil {
		log.Println(err)
	}
}

// setCurrencyHandler 实现用户手动选择货币种类。请求路径：/setCurrency POST. 详见 header.html: 73
func (fe *frontendServer) setCurrencyHandler(w http.ResponseWriter, r *http.Request) {
	cur := r.FormValue("currency_code")                    // 自动从请求中提取名为 currency_code 的参数，无需关心请求方式是POST还是GET
	payload := validator.SetCurrencyPayload{Currency: cur} // 构造一个 SetCurrencyPayload 对象，包含用户选择的货币代码
	// 下面执行校验
	if err := payload.Validate(); err != nil {
		log.Infof("无效的货币代码 %q: %v", cur, err)
		renderHTTPError(log, r, w, fmt.Errorf("无效的货币代码 %q", cur), http.StatusBadRequest)
		return
	}
	log.WithField("当前货币", payload.Currency).WithField("原货币", currentCurrency(r)).Debug("正在切换货币种类")

	// 已经确认货币有效，把用户选择的货币代码写入 Cookie，设置过期时间
	http.SetCookie(w, &http.Cookie{
		Name:   cookieCurrency,
		Value:  payload.Currency,
		MaxAge: cookieMaxAge,
	})

	// 设置货币（Set Currency）是一个 Action（动作），用户提交后，你不仅要设置 Cookie，通常还要让页面跳转回原来的地方（Referer）或者首页，否则用户会看到一个空白页面。
	referer := r.Referer() // 获取 Referer 头，得到用户之前所在的页面 URL. 等价于 r.Header.Get("referer")
	if referer == "" {
		referer = baseUrl + "/" // 如果没有 Referer 头，就跳转到首页
	}
	http.Redirect(w, r, referer, http.StatusSeeOther) // 重定向用户回原来的页面。这里用303 See Other 状态码，表示请求已经被处理，用户应该使用 【GET】 方法访问 Referer URL 来查看结果。
	// 之所以不用 302 Found，是因为 302 在 HTTP/1.0 中定义为临时重定向，但在 HTTP/1.1 中被重新定义为【根据请求方法决定重定向方式】，如果原请求是 POST，302 会被一些浏览器错误地处理为【继续使用 POST 方法访问 Referer URL】
	// 这可能导致问题。而 303 See Other 明确表示无论原请求是什么方法，用户都应该使用 GET 方法访问 Referer URL，这样更符合我们的需求。
}

// viewCartHandler
// TODO viewCartHandler待完善
func (fe *frontendServer) viewCartHandler(w http.ResponseWriter, r *http.Request) {
	cart := []*pb.CartItem{
		{ProductId: "MAHOYOSSSS", Quantity: 111},
		{ProductId: "6E92ZMYYFZ", Quantity: 1222},
	}
	shippingCost, err := fe.getShippingQuote(r.Context(), cart, "JPY")
	log.Debugf("getShippingQuote的执行结果 %v", shippingCost)
	if err != nil {
		log.Errorf("[shippingcost]: error %v", err)
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

// sessionID 返回请求头的 Cookie 中的 sessionID
func sessionID(r *http.Request) string {
	v := r.Context().Value(ctxKeySessionID{})
	if v == nil {
		return ""
	}
	return v.(string)
}

// currentCurrency 从请求的 Cookie 中提取用户的货币,如果没有设置则返回默认货币。
// setCurrencyHandler 这里将用户选择的货币放入了Cookie
// 提取：c, _ := r.Cookie(cookieCurrency)
func currentCurrency(r *http.Request) string {
	c, _ := r.Cookie(cookieCurrency)
	if c != nil {
		return c.Value
	}
	return "JPY"
}

// chooseAd 从获取到的广告列表中随机选择一个广告返回给模版渲染
func (fe *frontendServer) chooseAd(ctx context.Context, ctxKeys []string, log logrus.FieldLogger) *pb.Ad {
	ads, err := fe.getAd(ctx, ctxKeys)
	if err != nil {
		log.Errorf("[chooseAd]无法获取广告: %v", err)
		return nil
	}

    res := ads[rand.Intn(len(ads))]
	log.Debugf("[chooseAd]从 %d 个广告中选择了广告: %v", len(ads), res)
	return res
}

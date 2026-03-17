package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/mux"
	pb "github.com/kznLeaf/curated-store/src/frontend/genproto"
	money "github.com/kznLeaf/curated-store/src/frontend/money"
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
	ctx := r.Context()
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)

	log.WithField("[homehandler]currency", currentCurrency(r)).Info("home")

	currencies, err := fe.getCurrencies(ctx)
	if err != nil {
		log.Infof("could not retrieve currencies: %v", err)
		renderHTTPError(log, r, w, errors.New("could not retrieve currencies"), http.StatusInternalServerError)
		return
	}

	products, err := fe.GetProducts(ctx)
	if err != nil {
		log.Infof("could not retrieve products: %v", err)
		renderHTTPError(log, r, w, errors.New("could not retrieve products"), http.StatusInternalServerError)
		return
	}

	type productView struct {
		Item  *pb.Product // Product (string)类型定义在 proto，包含一个Picture字段，用于显示产品图片
		Price *pb.Money
	}
	ps := make([]productView, len(products))
	for i, p := range products {
		price, err := fe.convertCurrency(ctx, p.GetPriceUsd(), currentCurrency(r))
		if err != nil {
			log.Infof("could not convert currency: %v", err)
			renderHTTPError(log, r, w, errors.New("could not convert currency"), http.StatusInternalServerError)
			return
		}
		ps[i] = productView{p, price}
	}

	cart, err := fe.getCart(ctx, sessionID(r))
	if err != nil {
		renderHTTPError(log, r, w, errors.New("could not retrieve cart"), http.StatusInternalServerError)
		return
	}

	// 设置平台信息
	var env = os.Getenv("ENV_PLATFORM") // 如果没有该环境变量，说明是local环境。GCP会设置该环境变量.
	if env == "" || !stringinSlice(validEnvs, env) {
		log.Infof("could not retrieve platform details: %v", err)
		env = "local"
	}

	addrs, err := net.LookupHost("metadata.google.internal.") // 查询域名的IP地址，只有内部的实例才能得到查询结果
	if err == nil && len(addrs) >= 0 {
		log.Debugf("Detected Google metadata server: %v, setting ENV_PLATFORM to GCP.", addrs)
		env = "gcp"
	}

	log.Debugf("Current environment: %s", env)
	plat = &platformDetails{}
	plat.setPlatformDetails(strings.ToLower(env))

	if err := templates.ExecuteTemplate(w, "home", injectCommonTemplateData(r, map[string]interface{}{
		"show_currency": true,
		"currencies":    currencies,
		"products":      ps,
		"banner_color":  os.Getenv("BANNER_COLOR"),
		"cart_size":     cartSize(cart),
		"ad":            nil, // FIXME home.html 里完全没有调用 {{ template "text_ad" }} 的代码，这里实际上是一个无效传入。
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
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	ctx := r.Context()

	id := mux.Vars(r)["id"]
	if id == "" {
		renderHTTPError(log, r, w, errors.New("product id not specified"), http.StatusBadRequest)
		return
	}
	// 添加调试日志
	log.WithField("id", id).WithField("currency", currentCurrency(r)).Debug("[producthandler]debug info:")

	product, err := fe.GetProduct(ctx, id)
	if err != nil {
		log.Infof("could not retrieve product: %v", err)
		renderHTTPError(log, r, w, errors.New("could not retrieve product"), http.StatusInternalServerError)
		return
	}

	currencies, err := fe.getCurrencies(ctx)
	if err != nil {
		log.Infof("could not retrieve currencies: %v", err)
		renderHTTPError(log, r, w, errors.New("could not retrieve currencies"), http.StatusInternalServerError)
		return
	}

	price, err := fe.convertCurrency(ctx, product.GetPriceUsd(), currentCurrency(r))
	if err != nil {
		log.Infof("could not convert currency: %v", err)
		renderHTTPError(log, r, w, errors.New("could not convert currency"), http.StatusInternalServerError)
		return
	}

	recommendations, err := fe.getRecommendations(ctx, sessionID(r), nil) // TODO 第三个参数为nil,也就是随机从所有产品中抽取
	if err != nil {
		log.Infof("could not retrieve recommendations: %v", err)
	}

	wrappedProduct := struct {
		Item  *pb.Product
		Price *pb.Money
	}{product, price}

	cart, err := fe.getCart(ctx, sessionID(r))
	if err != nil {
		renderHTTPError(log, r, w, errors.New("could not retrieve cart"), http.StatusInternalServerError)
		return
	}

	// 渲染 product.html 模板，把填充后的页面写入 r
	if err := templates.ExecuteTemplate(w, "product", injectCommonTemplateData(r, map[string]interface{}{
		"ad":              fe.chooseAd(ctx, product.Categories, log),
		"show_currency":   true,
		"product":         wrappedProduct,
		"currencies":      currencies,
		"recommendations": recommendations,
		"cart_size":       cartSize(cart), // TODO packingInfo
	})); err != nil {
		log.Println(err)
	}
}

// setCurrencyHandler 实现用户手动选择货币种类。请求路径：/setCurrency POST. 详见 header.html: 73
func (fe *frontendServer) setCurrencyHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)

	cur := r.FormValue("currency_code")                    // 自动从请求中提取名为 currency_code 的参数，无需关心请求方式是POST还是GET
	payload := validator.SetCurrencyPayload{Currency: cur} // 构造一个 SetCurrencyPayload 对象，包含用户选择的货币代码
	// 下面执行校验
	if err := payload.Validate(); err != nil {
		log.Infof("Invalid currency code %q: %v", cur, err)
		renderHTTPError(log, r, w, fmt.Errorf("invalid currency code %q", cur), http.StatusBadRequest)
		return
	}

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

// viewCartHandler 适用于 /cart GET HEAD请求
func (fe *frontendServer) viewCartHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	ctx := r.Context()

	log.Debug("view cart")

	currencies, err := fe.getCurrencies(ctx)
	if err != nil {
		renderHTTPError(log, r, w, fmt.Errorf("could not retrieve currencies: %v", err), http.StatusInternalServerError)
		return
	}

	cartItems, err := fe.getCart(ctx, sessionID(r))
	if err != nil {
		renderHTTPError(log, r, w, fmt.Errorf("could not retrieve cart items: %v", err), http.StatusInternalServerError)
		return
	}

	recommendations, err := fe.getRecommendations(ctx, sessionID(r), cartIDs(cartItems))
	if err != nil {
		// 获取推荐失败不应该影响用户查看购物车的体验，所以这里记录日志但不返回错误给用户
		log.WithField("error", err).Warn("failed to get product recommendations")
	}

	shippingCost, err := fe.getShippingQuote(ctx, cartItems, currentCurrency(r))
	if err != nil {
		renderHTTPError(log, r, w, fmt.Errorf("could not retrieve shipping quote: %v", err), http.StatusInternalServerError)
		return
	}

	// 构造一个 cartItemView 切片，包含每个购物车项的产品信息、数量和价格，供模版渲染调用
	type cartItemView struct {
		Item     *pb.Product
		Quantity int32
		Price    *pb.Money
	}
	items := make([]cartItemView, len(cartItems))
	totalPrice := &pb.Money{CurrencyCode: currentCurrency(r)} // 购物车中所有产品的总价

	for i, cartItem := range cartItems {
		p, err := fe.GetProduct(ctx, cartItem.GetProductId())
		if err != nil {
			renderHTTPError(log, r, w, fmt.Errorf("could not get product info: %v", err), http.StatusInternalServerError)
			return
		}

		price, err := fe.convertCurrency(ctx, p.GetPriceUsd(), currentCurrency(r))
		if err != nil {
			renderHTTPError(log, r, w, fmt.Errorf("could not convert currency: %v", err), http.StatusInternalServerError)
			return
		}

		// 计算每个购物车项的价格 = 产品价格 * 数量。这里直接把 price 乘以数量，得到总价。
		// 注意 price 是一个 Money 对象，不能直接乘以数量，需要调用 money 包里的函数来实现。
		multiprice := money.MultiplySlow(price, uint32(cartItem.GetQuantity()))
		items[i] = cartItemView{
			Item:     p,
			Quantity: cartItem.GetQuantity(),
			Price:    multiprice,
		}
		totalPrice = money.Must(money.Sum(totalPrice, multiprice)) // 累加到总价
	}
	// 加上运费
	totalPrice = money.Must(money.Sum(totalPrice, shippingCost))
	year := time.Now().Year()

	if err := templates.ExecuteTemplate(w, "cart", injectCommonTemplateData(r, map[string]interface{}{
		"currencies":       currencies,
		"recommendations":  recommendations,
		"shipping_cost":    shippingCost,
		"cart_size":        cartSize(cartItems),
		"items":            items,
		"total_cost":       totalPrice,
		"expiration_years": []int{year, year + 1, year + 2, year + 3, year + 4}, // 给购物车页面里信用卡到期年份下拉菜单提供选项数据
	})); err != nil {
		log.Error(err)
	}
}

// addToCartHandler 适用于 /cart POST 请求，处理用户添加商品到购物车的请求
func (fe *frontendServer) addToCartHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	ctx := r.Context()

	productId := r.FormValue("product_id") // 从请求中提取 product_id 参数
	quantity, _ := strconv.ParseUint(r.FormValue("quantity"), 10, 32)

	payload := validator.AddToCartPayload{
		ProductID: productId,
		Quantity:  quantity,
	}
	if err := payload.Validate(); err != nil {
		log.WithField("validation_error", err).Warn("add to cart validation failed")
		renderTopValidationPopup(r, w, validator.ValidationErrorResponse(err), http.StatusUnprocessableEntity)
		return
	}
	log.WithField("product", payload.ProductID).WithField("quantity", payload.Quantity).Debug("adding to cart")

	p, err := fe.GetProduct(ctx, payload.ProductID)
	if err != nil {
		renderHTTPError(log, r, w, fmt.Errorf("could not retrieve product: %v", err), http.StatusInternalServerError)
		return
	}

	err = fe.insertCart(ctx, sessionID(r), p.GetId(), int32(quantity))
	if err != nil {
		renderHTTPError(log, r, w, fmt.Errorf("could not insert cart item: %v", err), http.StatusInternalServerError)
		return
	}

	// TODO Referer校验，避免开放重定向漏洞

	referer := r.Referer()
	if referer == "" {
		referer = baseUrl + "/"
	}

	// 校验 referer，防止开放重定向。如果校验失败，就直接回到首页
	if u, parseErr := url.Parse(referer); parseErr != nil || (u.Host != "" && u.Host != r.Host) {
		referer = baseUrl + "/"
	}

	http.Redirect(w, r, referer, http.StatusSeeOther)
}

func (fe *frontendServer) emptyCartHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	ctx := r.Context()

	log.Debug("empty cart")
	err := fe.emptyCart(ctx, sessionID(r))
	if err != nil {
		renderHTTPError(log, r, w, fmt.Errorf("failed to empty cart: %v", err), http.StatusInternalServerError)
		return
	}
	// TODO Referer校验，避免开放重定向漏洞
	referer := r.Referer()
	if referer == "" {
		referer = baseUrl + "/"
	}

	// 校验 referer，防止开放重定向。如果校验失败，就直接回到首页
	if u, parseErr := url.Parse(referer); parseErr != nil || (u.Host != "" && u.Host != r.Host) {
		referer = baseUrl + "/"
	}
	http.Redirect(w, r, referer, http.StatusSeeOther)
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
		log.Errorf("[chooseAd]failed to fetch ads: %v", err)
		return nil
	}

	res := ads[rand.Intn(len(ads))]
	// log.Debugf("[chooseAd] %d : %v", len(ads), res)
	return res
}

// cartIDs 从购物车项列表中提取产品ID列表，供 getRecommendations 调用
func cartIDs(cartItems []*pb.CartItem) []string {
	ids := make([]string, len(cartItems))
	for i, item := range cartItems {
		ids[i] = item.GetProductId()
	}
	return ids
}

// cartSize 计算购物车中商品的总数量，供模版渲染调用
func cartSize(c []*pb.CartItem) int {
	cartSize := 0
	for _, item := range c {
		cartSize += int(item.GetQuantity())
	}
	return cartSize
}

func (fe *frontendServer) insertCart(ctx context.Context, userID, productID string, quantity int32) error {
	_, err := pb.NewCartServiceClient(fe.cartSvcConn).AddItem(ctx, &pb.AddItemRequest{
		UserId: userID,
		Item: &pb.CartItem{
			ProductId: productID,
			Quantity:  quantity,
		},
	})
	return err
}

func (fe *frontendServer) emptyCart(ctx context.Context, userID string) error {
	_, err := pb.NewCartServiceClient(fe.cartSvcConn).EmptyCart(ctx, &pb.EmptyCartRequest{
		UserId: userID,
	})
	return err
}

func (fe *frontendServer) placeOrderHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	log.Debug("placing order")
	ctx := r.Context()
	// 解析表单数据
	var (
		email         = r.FormValue("email")
		streetAddress = r.FormValue("street_address")
		zipCode, _    = strconv.ParseInt(r.FormValue("zip_code"), 10, 32)
		city          = r.FormValue("city")
		state         = r.FormValue("state")
		country       = r.FormValue("country")
		ccNumber      = r.FormValue("credit_card_number")
		ccMonth, _    = strconv.ParseInt(r.FormValue("credit_card_expiration_month"), 10, 32)
		ccYear, _     = strconv.ParseInt(r.FormValue("credit_card_expiration_year"), 10, 32)
		ccCVV, _      = strconv.ParseInt(r.FormValue("credit_card_cvv"), 10, 32)
	)
	// 创建 PlaceOrderPayload 对象
	payload := validator.PlaceOrderPayload{
		Email:         email,
		StreetAddress: streetAddress,
		ZipCode:       zipCode,
		City:          city,
		State:         state,
		Country:       country,
		CcNumber:      ccNumber,
		CcMonth:       ccMonth,
		CcYear:        ccYear,
		CcCVV:         ccCVV,
	}
	// 验证表单数据，验证规则：
	if err := payload.Validate(); err != nil {
		log.WithField("validation_error", err).Warn("place order validation failed")
		renderTopValidationPopup(r, w, validator.ValidationErrorResponse(err), http.StatusUnprocessableEntity)
		return
	}

	order, err := pb.NewCheckoutServiceClient(fe.checkoutSvcConn).PlaceOrder(
		ctx, &pb.PlaceOrderRequest{
			Email: payload.Email,
			CreditCard: &pb.CreditCardInfo{
				CreditCardNumber:          payload.CcNumber,
				CreditCardExpirationMonth: int32(payload.CcMonth),
				CreditCardExpirationYear:  int32(payload.CcYear),
				CreditCardCvv:             int32(payload.CcCVV)},
			UserId:       sessionID(r),
			UserCurrency: currentCurrency(r),
			Address: &pb.Address{
				StreetAddress: payload.StreetAddress,
				City:          payload.City,
				State:         payload.State,
				ZipCode:       int32(payload.ZipCode),
				Country:       payload.Country},
		})
	if err != nil {
		renderHTTPError(log, r, w, fmt.Errorf("failed to complete the order: %v", err), http.StatusInternalServerError)
		return
	}
	log.WithField("order", order.GetOrder().GetOrderId()).Info("order placed")

	recommendations, recommendationErr := fe.getRecommendations(ctx, sessionID(r), nil)
	if recommendationErr != nil {
		log.WithField("error", recommendationErr).Warn("could not retrieve recommendations")
	}

	// 计算总支付金额
	totalPaid := order.GetOrder().GetShippingCost()
	for _, v := range order.GetOrder().GetItems() {
		multPrice := money.MultiplySlow(v.GetCost(), uint32(v.GetItem().GetQuantity()))
		totalPaid = money.Must(money.Sum(totalPaid, multPrice))
	}
	// 获取可用货币列表
	currencies, err := fe.getCurrencies(ctx)
	if err != nil {
		renderHTTPError(log, r, w, fmt.Errorf("could not retrieve currencies: %v", err), http.StatusInternalServerError)
		return
	}

	// 渲染订单确认页
	if err := templates.ExecuteTemplate(w, "order", injectCommonTemplateData(r, map[string]interface{}{
		"show_currency":   false,
		"currencies":      currencies,
		"order":           order.GetOrder(),
		"total_paid":      &totalPaid,
		"recommendations": recommendations,
	})); err != nil {
		log.Println(err)
	}

}

func (fe *frontendServer) assistantHandler(w http.ResponseWriter, r *http.Request) {

	currencies, err := fe.getCurrencies(r.Context())
	if err != nil {
		renderHTTPError(log, r, w, fmt.Errorf("could not retrieve currencies: %v", err), http.StatusInternalServerError)
		return
	}

	if err := templates.ExecuteTemplate(w, "assistant", injectCommonTemplateData(r, map[string]interface{}{
		"show_currency": false,
		"currencies":    currencies,
	})); err != nil {
		log.Println(err)
	}

}

func (fe *frontendServer) loginHandler(w http.ResponseWriter, r *http.Request) {
	log.Info("对喽")
	log.WithFields(logrus.Fields{"Form": r.Form}).Info("登录表单数据")

}

// renderTopValidationPopup 在用户输入有误时，渲染一个包含错误信息的弹窗，并引导用户返回之前的页面继续操作
func renderTopValidationPopup(r *http.Request, w http.ResponseWriter, err error, _ int) {
	log.WithField("validation_error", err).Warn("form validation failed")

	referer := r.Referer()
	if referer == "" {
		referer = baseUrl + "/"
	}

	// 校验 referer，防止开放重定向
	if u, parseErr := url.Parse(referer); parseErr != nil || (u.Host != "" && u.Host != r.Host) {
		referer = baseUrl + "/"
	}

	// 在 URL 中添加 validation_error 参数，值为错误信息，这样前端页面就可以读取这个参数并显示弹窗
	redirectURL, _ := url.Parse(referer)
	q := redirectURL.Query()
	q.Set("validation_error", err.Error())
	redirectURL.RawQuery = q.Encode()

	http.Redirect(w, r, redirectURL.String(), http.StatusSeeOther)
}

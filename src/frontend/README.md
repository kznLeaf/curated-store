# frontend

该服务使用 go 的 `template` 包来渲染 html 页面模板，然后返回给浏览器进行渲染。前端页面相关的资源位于`static/`和`templates`。将这些静态资源引入项目的方式不同：

1. `templates/` — 通过 `html/template` 包在启动时加载到内存

```go
var (
    templates = template.Must(template.New("").
        Funcs(template.FuncMap{
            "renderMoney":        renderMoney,
            "renderCurrencyLogo": renderCurrencyLogo,
        }).ParseGlob("templates/*.html"))  // ← 关键在这里
)
```

`ParseGlob`将所有的`.html`文件加载进内存，后续每次渲染只需要调用`templates.ExecuteTemplate(w, "xxxx", data)`，不再读磁盘。

2. `statc/`注册了一条路由

```go
r.PathPrefix(baseUrl + "/static/").Handler(http.StripPrefix(baseUrl + "/static/", http.FileServer(http.Dir("./static/"))))
```

将所有`baseUrl + "/static/"`开头的请求去掉前缀之后，交给静态文件服务器`./static/`寻找该资源。

运行下面的命令，将 frontend 的8080端口转发到本机8080端口：

```bash
kubectl port-forward deployment/frontend 8080:8080
```

在浏览器中访问 localhost:8080
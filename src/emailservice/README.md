# currencyservice

邮件服务会在客户成功结账后生成并发送订单确认邮件。它从结账服务接收订单详情，并使用 Jinja2 模板格式化 HTML 邮件内容。

![structure](README-image.png)

目前实现了一个虚拟的邮件服务，收到请求之后只会打印日志，并返回空的响应。
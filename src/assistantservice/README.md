## assistantservice

用户请求路径

```
设置 ENABLE_ASSISTANT=true
    → 导航栏出现魔法棒图标
    → 点击图标进入 /assistant 页面
    → 在输入框输入消息（可选上传图片）点击 Send
    → 前端 JS POST /bot
    → 后端转发给 Shopping Assistant 微服务
    → 返回结果渲染在聊天窗口
```


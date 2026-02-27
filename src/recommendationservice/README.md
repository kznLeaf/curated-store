```bash
python -m venv venv        # 创建虚拟环境
source venv/bin/activate   # 激活虚拟环境，如果使用了.envrc 则可以忽略
pip install pip-tools      # 包含了 pip-compile 命令
pip install grpcio-tools   # 用于生成 proto 产物
pip-compile requirements.in
pip install -r requirements.txt # 每次更新 requirements.in 之后都要用这条命令来刷新
```

为了避免每次进入项目时都手动激活虚拟环境，安装[direnv](https://direnv.net/)：

```bash
sudo pacman -S direnv
```

然后在本服务的目录下运行`direnv allow`以启用`.envrc`文件。

---

`grpcio-health-checking` 这个包没有附带类型存根（`.pyi` 文件），Pylance 找不到源码来做类型推断，所以会提示“无法从源解析导入“grpc_health.v1.health_pb2”。

gRPC 的交叉版本运行保证，参见：https://protobuf.dev/support/cross-version-runtime-guarantee/，核心是运行时的版本不能比生成的protoc代码的版本更旧。




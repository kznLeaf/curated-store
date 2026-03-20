## Setup Guide

### Create GitHub OAuth App
Register a new OAuth application [here](https://github.com/settings/applications/new). 

![](./image.png)

Ensure the **Authorization callback URL** matches your Dex configuration (e.g., `https://localhost:5556/dex/callback`).

### Generate TLS Assets

Running Dex with HTTPS enabled requires a valid SSL certificate. Generate the necessary SSL certificates and CA files by running:

```bash
./src/frontend/gencert.sh
```

The assets will be generated in the `ssl/` directory.

### Run Frontend Service

Start the local frontend client using the generated CA:

```bash
go run . -issuer-root-ca ./ssl/ca.pem
```

### Configure Kubernetes Secrets

First, run `gencert.sh` to generate necessary files. 

Then, run `./set-up-secrets.sh` before you deploy resources, which includes two steps:

1. Create TLS Secret in `default` namespace:

```bash
kubectl -n default create secret tls dex.example.com.tls \
  --cert=./cert.pem \
  --key=./key.pem
```

The CA file will be mounted as Secret in `set-up-secrets.sh` so that the frontendservice can access it.

2. Create GitHub Credentials Secret in `default` namespace, and before that you must ensure `GITHUB_CLIENT_ID` and `GITHUB_CLIENT_SECRET` have been added as your shell environment varibles.

```bash
kubectl -n default create secret generic github-client \
  --from-literal=client-id=$GITHUB_CLIENT_ID \
  --from-literal=client-secret=$GITHUB_CLIENT_SECRET
```

## 建立过程

**时空错位**

集群启动后，frotnend和dex都是集群内部的一个pod，分别端口转发到本地的8080和5556端口。frontend需要访问dex的issuer URL进行OIDC的自动发现。由于frontend和dex都位于集群内部，故将issuer URL定义为

```
https://dex:5556/dex
```

但是这样的话，点击浏览器之后，跳转的地址就是`https://dex:5556/dex/auth?client_id=...&...`，会访问出错。解决方法：修改本机的hosts文件，将`dex`指向`localhost`. 这是本地部署的解决方案，如果是云端，需要给dex分配一个专门的二级域名，例如`dex.example.com`，也就是issuer URL。

这还没完，这里dex用到的证书是用我们自己的CA签发的，所以需要把CA的证书加入到系统。




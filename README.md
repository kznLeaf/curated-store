This project is restructured from [microservices-demo by Google](https://github.com/GoogleCloudPlatform/microservices-demo). 变动包括：

- 将原项目除了python以外的后端服务全部用Go重写
- 完全实现Opentelementry链路追踪
- 增加了 Jaeger 的k8s配置，在本地部署的情况下即可查看链路追踪的Web UI
- 将原项目的python后端的包管理从pip换到uv
- Dockerfile中已经尽可能配置了中国大陆的镜像源，只需要提前拉取基础镜像就可以回避网络问题
- 修复原项目中存在的bug,重构部分代码，如优雅关机等，更符合软件开发规范

## Quickstart(local)

### 开始前的准备

本项目在Linux平台上运行一个k8s集群。本地k8s集群主要有`minikube`和`kind`两种方式，这里我推荐使用`kind`.

确保你已经安装`skaffold` `kind` `kubectl` `docker` `docker-buildx`。对于 Archlinux 用户，这些软件包都可以用pacman快速获取。

创建一个集群：

```bash
kind create cluster --name mydemo
```

指定 skaffold 操作本地集群

```bash
skaffold config set --global local-cluster true
```

将`kubectl`的上下文设置为该集群

```bash
kubectl cluster-info --context kind-test
```

### 部署

See: https://skaffold.dev/docs/cleanup/ , for a kind cluster, in order to avoid images piling up, run:

```bash
skaffold dev --no-prune=false --cache-artifacts=false
```

If piling up has already occurred, you can run the command below to prune the unused images:

```bash
docker rmi $(docker images --format "{{.Repository}}:{{.Tag}}" | grep -E "service|frontend|loadgenerator|skaffold")
```

<details>
  <summary>中国大陆情况的部署方法</summary>

由于中国大陆以"内容未经审查"为借口，长期以来借助[GFW](https://en.wikipedia.org/wiki/Great_Firewall)对境外网站执行无差别拦截，docker源站也受到波及。由此简中互联网产生了一系列以“Docker镜像站”为主要盈利手段的劣质网站，如轩辕镜像站。然而，对于个人用户而言，docker hub绝不应该，也绝不能成为一项付费服务。我绝不推荐使用此类收费镜像站来拉取docker镜像，这属于助纣为虐；更何况轩辕镜像站提供的镜像列表本身就有**严重残缺**，例如本项目用到的 alpine 平台镜像就无法从中拉取，丝毫不值得为此花费一分钱。

在这一步可能报无法拉取镜像的错误，例如：

```
Error: container redis is waiting to start: redis:alpine can't be pulled.
```

kind的一大优势就在于：它是基于本地Docker daemon运行的集群。如果集群无法拉取镜像，那就可以直接用本地docker拉取镜像，打tag，然后手动加载到集群中。类似的网络问题都可以这样来解决。

</details>


# Github Workflows

## ci-main-build

1. `code-tests`运行单元测试
   1. Set up Go
   2. GO Unit Tests
2. `skaffold-check`测试能否成功构建镜像。步骤如下：
   1. 安装`skaffold`
   2. 检查配置是否正确，这里有两步：
      - `skaffold diagnose --check`静态校验配置文件，不产生镜像，速度很快
      - `skaffold build --push=false`调用配置文件中定义的构建器（比如Docker）进行镜像构建，只是构建完毕后不会推送到远程仓库


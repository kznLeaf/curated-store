# components的处理办法

## 检查合并后的配置文件

在项目根目录下运行

```bash
kubectl kustomize kubernetes-manifests 
```

相当于在`kubernetes-manifests/`目录下运行`kustomize build`

## 使用 `kustomize edit` 添加组件

例如启用 **Cymbal Shops Branding**：

```bash
kustomize edit add component components/cymbal-branding
```

再叠加 Google Cloud Operations：

```bash
kustomize edit add component components/google-cloud-operations
```

---

## 部署组件

你可以先渲染：

```bash
kubectl kustomize .
```

然后部署：

```bash
kubectl apply -k .
```

示例 `kustomization.yaml`：

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- base
components:
- components/cymbal-branding
- components/google-cloud-operations
```

---

## 使用远程 Kustomize 目标

Kustomize 支持引用 GitHub 等远程资源：

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- github.com/GoogleCloudPlatform/microservices-demo/kustomize/base
components:
- github.com/GoogleCloudPlatform/microservices-demo/kustomize/components/cymbal-branding
- github.com/GoogleCloudPlatform/microservices-demo/kustomize/components/google-cloud-operations
```

更多信息见 [Kustomize remote targets](https://github.com/kubernetes-sigs/kustomize/blob/master/examples/remoteBuild.md)。


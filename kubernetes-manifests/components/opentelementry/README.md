This project provides two ways to observe Traces:

* **Deploy Jaeger within the cluster**: The collector sends traces to Jaeger, and you can observe them using the Web UI provided by Jaeger. This is suitable for local deployment.
* **Send traces to Google Cloud**: The collector sends traces to Google Cloud, which is suitable for cloud-based deployment.

The only difference between the two methods lies in the **exporter** configuration within the otel-collector. For details, refer to `otel-collector-jaeger.yaml` and `otel-collector.yaml`. Regardless of the method used, a single **Collector** service exists in the cluster. Each service sends its collected telemetry data to this Collector, which then decides whether to export it to Jaeger or Google Cloud.

---

## Integration with Jaeger

Refer to the configuration file [otel-collector-jaeger.yaml](https://www.google.com/search?q=./otel-collector-jaeger.yaml). Jaeger is deployed as a pod within the cluster and exposes its Web UI on port **16686**. To access this page in a local browser, set up port forwarding first:

```bash
kubectl port-forward deployment/opentelemetrycollector 16686:16686
```

Then, visit http://localhost:16686 in your browser, and you will see something similar to the following:

![jaeger](./jaeger.png)

---

## Integration with Google Cloud Operations

By default, the detection features for [Google Cloud Operations](https://cloud.google.com/products/operations) (including Monitoring/Stats, Tracing, and Profiler) are **disabled** in the Online Boutique deployment. This means that even if you run this application on [GKE](https://cloud.google.com/kubernetes-engine), tracing information will not be exported to [Google Cloud Trace](https://cloud.google.com/trace).

To re-enable Google Cloud Operations detection, the simplest way is to enable the built-in kustomize component. This component enables tracing and metrics and adds an [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/) deployment to collect and forward data to the corresponding Google Cloud backends.

Run the following command in the `kustomize/` directory at the root of the repository:

```bash
kustomize edit add component components/google-cloud-operations
```

This updates the `kustomize/kustomization.yaml` file to look similar to this:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- base
components:
- components/google-cloud-operations
```

You can render these configurations locally by running `kubectl kustomize .` or deploy them using `kubectl apply -k .`.

### Enable Google Cloud APIs

Ensure the relevant Google APIs are enabled in your project:

```bash
PROJECT_ID=<your-gcp-project-id>
gcloud services enable \
    monitoring.googleapis.com \
    cloudtrace.googleapis.com \
    cloudprofiler.googleapis.com \
    --project ${PROJECT_ID}
```

### Grant IAM Roles

Grant the following IAM roles to your Google Service Account (GSA):

```bash
PROJECT_ID=<your-gcp-project-id>
GSA_NAME=<your-gsa>

gcloud projects add-iam-policy-binding ${PROJECT_ID} \
  --member "serviceAccount:${GSA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role roles/cloudtrace.agent

gcloud projects add-iam-policy-binding ${PROJECT_ID} \
  --member "serviceAccount:${GSA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role roles/monitoring.metricWriter
  
gcloud projects add-iam-policy-binding ${PROJECT_ID} \
  --member "serviceAccount:${GSA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role roles/cloudprofiler.agent
```

**Note**: Currently, only **Tracing** is supported; support for Metrics and further features is coming soon.

---

## Changes Applied

Once the kustomize component is enabled, most services will receive a configuration patch similar to the following:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: productcatalogservice
spec:
  template:
    spec:
      containers:
        - name: server
          env:
          - name: COLLECTOR_SERVICE_ADDR
            value: "opentelemetrycollector:4317"
          - name: ENABLE_STATS
            value: "1"
          - name: ENABLE_TRACING
            value: "1"
```

This patch sets environment variables to enable the export of metrics and traces and specifies how the service connects to the newly deployed Collector.

---

## OpenTelemetry Collector

Currently, this component adds a single collector service which collects traces and metrics from individual services and forwards them to the appropriate Google Cloud backend.

![Collector Architecture Diagram](collector-model.png)

If you wish to experiment with different backends, you can modify the appropriate lines in [otel-collector.yaml](otel-collector.yaml) to export traces or metrics to a different backend.  See the [OpenTelemetry docs](https://opentelemetry.io/docs/collector/configuration/) for more details.

---

## Workload Identity

If you are running this on GKE with [Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity) enabled, traces might fail to export, and you may see `PermissionDenied` errors in the `opentelemetrycollector` pod logs.

In this case, you need to associate the Kubernetes [Service Account](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/) (`default/default`) with your Google Cloud Service Account.

1. **Get the GSA Email**:
```bash
gcloud iam service-accounts list
```

2. **Allow the KSA to impersonate the GSA**:
```bash
gcloud iam service-accounts add-iam-policy-binding ${GSA_EMAIL} \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:${PROJECT_ID}.svc.id.goog[default/default]"
```

3. **Annotate the Kubernetes Service Account**:
```bash
kubectl annotate serviceaccount default \
  iam.gke.io/gcp-service-account=${GSA_EMAIL}
```

4. **Restart the Collector deployment**:
```bash
kubectl rollout restart deployment opentelemetrycollector
```

For methods on analyzing traces, please refer to [this Google Cloud blog post](https://www.google.com/search?q=https://cloud.google.com/blog/products/devops-sre/using-cloud-trace-and-cloud-logging-for-root-cause-analysis%3Fhl%3Den).

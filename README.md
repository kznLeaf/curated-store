Restructured from [microservices-demo by Google](https://github.com/GoogleCloudPlatform/microservices-demo).

See: https://skaffold.dev/docs/cleanup/ , for kind cluster, run `skaffold dev --no-prune=false --cache-artifacts=false` to avoid images piling up. If that has already occurred, run:

```bash
docker rmi $(docker images --format "{{.Repository}}:{{.Tag}}" | grep -E "service|frontend|loadgenerator|skaffold")
```
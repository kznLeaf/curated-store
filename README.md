Restructured from [microservices-demo by Google](https://github.com/GoogleCloudPlatform/microservices-demo).

## Quickstart(local)

See: https://skaffold.dev/docs/cleanup/ , for a kind cluster, in order to avoid images piling up, run:

```bash
skaffold dev --no-prune=false --cache-artifacts=false
```

If that has already occurred, run:

```bash
docker rmi $(docker images --format "{{.Repository}}:{{.Tag}}" | grep -E "service|frontend|loadgenerator|skaffold")
```
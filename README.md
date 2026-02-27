# Package Universe

A small Golang app to handle various types of software packages

## Container Registry

The server includes an OCI-compatible container registry. Start it with:

```bash
make run
```

### Pushing images on macOS with Docker Desktop

On macOS, Docker Desktop runs the Docker daemon inside a Linux VM. `localhost` inside that VM refers to the VM itself, not the macOS host, so `docker push localhost:8080/...` will fail with `context deadline exceeded`.

Use `host.docker.internal` instead, which resolves to the host machine from within the VM:

```bash
docker tag nginx:latest host.docker.internal:8080/test/nginx:latest
docker push host.docker.internal:8080/test/nginx:latest
```

Since the registry runs over plain HTTP, you must add it to Docker's insecure registries. Edit `~/.docker/daemon.json` (or Docker Desktop Settings > Docker Engine):

```json
{
  "insecure-registries": [
    "host.docker.internal:8080"
  ]
}
```

Restart Docker Desktop after making the change.

### Verifying

```bash
# Check the registry is responding
curl http://localhost:8080/v2/

# List tags for a pushed image
curl http://localhost:8080/v2/test/nginx/tags/list
```

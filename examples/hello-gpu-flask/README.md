# Hello GPU HTTP API

A simple GPU HTTP API that just returns the output of `nvidia-smi` using Flask.

# Run locally

Requirements:

- [nvidia-docker](#)
- An Nvidia GPU compatible with CUDA

### 1 - Build the Docker image:

```bash
docker build hello-gpu .
```

### 2 - Start the server:

```bash
docker --runtime=nvidia-docker -p 8080:80 --rm run hello-gpu
```

### 3 - Test the API, for example using cURL:

```
curl http://localhost:8080
```

# Run as remote serverless API

Requirements:

- [Kuda CLI](#) pointing to a Kubernetes cluster with [Kuda](#).

## 1 - Initialize

```bash
# Replace with your cluster's ingress IP.
export cluster_ip="XX.XX.XX.XX"

# Replace with your docker container registry such as docker.io, gcr.io..etc.
# Docker must have write access to the registry.
export docker_registry="docker.io/username"
```

Then generate the configuration files using `kuda init`:

```bash
kuda init \
   -d $docker_registry/hello-gpu \
   http://hello-gpu.default.$cluster_ip.xip.io
```

## 2 - Dev

Run the API remotely in dev mode using:

```bash
kuda dev
```

Depending on your configuration, the whole process could take a while.
But once the image has been built, pushed, provisioned, deployed & started,
you should start seeing the startup logs from the Flask debug server.

You can then call the API, for example using cURL:

```bash
curl http://hello-gpu.default.$cluster_ip.xip.io
```

💡You can try to update the code in `main.py` while `dev`
is running and the remote API should automatically synchronize & reload
with the new changes.

## 2 - Deploy

Once you're happy with your API, you can deploy the production build using:

```bash
kuda deploy
```
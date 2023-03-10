# gtctl

## Overview

`gtctl`(`g-t-control`) is a command-line tool for managing the [GreptimeDB](https://github.com/GrepTimeTeam/greptimedb) cluster. gtctl is the **All-in-One** binary that integrates multiple operations of the GreptimeDB cluster.

<p align="center">
<img alt="screenshot" src="./docs/images/screenshot.png" width="800px">
</p>

## One-line installation

```console
curl -fsSL https://raw.githubusercontent.com/greptimeteam/gtctl/develop/hack/install.sh | sh
```

After downloading, the `gtctl` will be in the current directory.

## Getting Started

### Prerequisites

- **Kubernetes 1.18 or higher version is required**

  You can use the [`kind`](https://kind.sigs.k8s.io/) to create your own Kubernetes cluster:

  ```console
  kind create cluster
  ```

### Cluster Operations

Create your own GreptimeDB cluster and etcd cluster:

```console
gtctl cluster create mycluster -n default
```

After creating, the whole GreptimeDB cluster will start in `default` namespace:

```console
# Get the cluster.
gtctl cluster get mycluster -n default

# List all clusters.
gtctl cluster list
```

You can use `kubectl port-forward` command to forward frontend requests:

```console
kubectl port-forward svc/mycluster-frontend 4002:4002 > connections.out &
```

Use your `mysql` client to connect to your cluster:

```console
mysql -h 127.0.0.1 -P 4002
```

If you want to delete the cluster, you can:

```console
# Delete the cluster.
gtctl cluster delete mycluster -n default

# Delete the cluster, including etcd cluster.
gtctl cluster delete mycluster -n default --tear-down-etcd
```

### Dry Run Mode

`gtctl` provides `--dry-run` option in cluster creation. If a user executes the command with `--dry-run`, `gtctl` will output the manifests content without applying them:

```console
gtctl cluster create mycluster -n default --dry-run
```

### Experimental Feature

You can use the following commands to scale (or down-scale) your cluster:

```console
# Scale datanode to 3 replicas.
gtctl cluster scale <your-cluster> -n <your-cluster-namespace> -c datanode --replicas 3

# Scale frontend to 5 replicas.
gtctl cluster scale <your-cluster> -n <your-cluster-namespace> -c frontend --replicas 5
```

### Specify the image registry

`gtctl` uses DockerHub as the default image registry and also supports specifying image registry when creating a cluster with the `--image-registry` option (the UCloud image registry mirror `uhub.service.ucloud.cn` is now available).

中国用户可使用如下命令创建集群：

```console
gtctl cluster create mycluster --image-registry=uhub.service.ucloud.cn
```

## Development

- Compile the project

  ```console
  make
  ```

  Then the `gtctl` will be generated in `bin/`.

## License

`gtctl` uses the [Apache 2.0 license](./LICENSE) to strike a balance between open contributions and allowing you to use the software however you want.


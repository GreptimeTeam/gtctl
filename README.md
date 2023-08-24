# gtctl

[![codecov](https://codecov.io/github/GreptimeTeam/gtctl/branch/develop/graph/badge.svg?token=287NUSEH5D)](https://app.codecov.io/github/GreptimeTeam/gtctl/tree/develop)
[![license](https://img.shields.io/github/license/GreptimeTeam/gtctl)](https://github.com/GreptimeTeam/gtctl/blob/develop/LICENSE)
[![report](https://goreportcard.com/badge/github.com/GreptimeTeam/gtctl)](https://goreportcard.com/report/github.com/GreptimeTeam/gtctl)

## Overview

`gtctl`(`g-t-control`) is a command-line tool for managing the [GreptimeDB](https://github.com/GrepTimeTeam/greptimedb) cluster. `gtctl` is the **All-in-One** binary that integrates multiple operations of the GreptimeDB cluster.

<p align="center">
<img alt="screenshot" src="./docs/images/screenshot.png" width="800px">
</p>

## Installation

### Binary

Download the binary using onel-line command:

```console
curl -fsSL https://raw.githubusercontent.com/greptimeteam/gtctl/develop/hack/install.sh | sh
```

After downloading, the `gtctl` will be in the current directory.

### Homebrew (macOS)

On macOS, `gtctl` is available via Homebrew:

```console
brew tap greptimeteam/greptime
brew install gtctl
```

### From Source

If you already have the [Go](https://go.dev/doc/install) installed, you can run `make` command under this project to install `gtctl`:

```console
make gtctl
```

After building, the `gtctl` will be generated in `./bin/`.

## Getting Started

### Playground

The **fatest** way to experience the GreptimeDB cluster is to use the playground:

```console
gtctl playground
```

The `playground` will deploy the minimal GreptimeDB cluster on your environment in bare-metal mode.

### Deploy in Bare-Metal Environment

You can deploy the GreptimeDB cluster on a bare-metal environment by the following simple command:

```console
gtctl cluster create mycluster --bare-metal
```

It will create all the meta information on `${HOME}/.gtctl`.

If you want to do more configurations, you can use the yaml format config file:

```console
gtctl cluster create mycluster --bare-metal --config <your-config-file>
```

You can refer to the example [`cluster.yaml`](./examples/bare-metal/cluster.yaml) and [`cluster-with-local-artifacts.yaml`](./examples/bare-metal/cluster-with-local-artifacts.yaml).

### Deploy in Kubernetes

#### Prerequisites

- **Kubernetes 1.18 or higher version is required.**

  You can use the [`kind`](https://kind.sigs.k8s.io/) to create your own Kubernetes cluster:

  ```console
  kind create cluster
  ```

#### Cluster Operations

Create your own GreptimeDB cluster and etcd cluster:

```console
gtctl cluster create mycluster -n default
```

After creating, the whole GreptimeDB cluster will start in the `default` namespace:

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

#### Dry Run Mode

`gtctl` provides `--dry-run` option in cluster creation. If a user executes the command with `--dry-run`, `gtctl` will output the manifests content without applying them:

```console
gtctl cluster create mycluster -n default --dry-run
```

#### Experimental Feature

You can use the following commands to scale (or down-scale) your cluster:

```console
# Scale datanode to 3 replicas.
gtctl cluster scale <your-cluster> -n <your-cluster-namespace> -c datanode --replicas 3

# Scale frontend to 5 replicas.
gtctl cluster scale <your-cluster> -n <your-cluster-namespace> -c frontend --replicas 5
```

#### Specify the image registry

`gtctl` uses DockerHub as the default image registry and also supports specifying image registry when creating a cluster with the `--image-registry` option (the AliCloud image registry mirror `greptime-registry.cn-hangzhou.cr.aliyuncs.com` is now available).

中国用户可使用如下命令创建集群：

```console
gtctl cluster create mycluster --image-registry=greptime-registry.cn-hangzhou.cr.aliyuncs.com
```

## Development

There are many useful tools provided through Makefile, you can simply run `make help` to get more information:

- Compile the project

  ```console
  make
  ```

  Then the `gtctl` will be generated in `./bin/`.


- Run the unit test

  ```console
  make test
  ```

- Run the e2e test

  ```console
  make e2e
  ```

## License

`gtctl` uses the [Apache 2.0 license](./LICENSE) to strike a balance between open contributions and allowing you to use the software however you want.

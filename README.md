# gtctl

## Overview

gtctl(`g-t-control`) is a command-line tool for managing [GreptimeDB](https://github.com/GrepTimeTeam/greptimedb) cluster. gtctl is the **All-in-One** binary that integrates multiple operations of GreptimeDB cluster.

![screenshot](docs/images/screenshot.png)

## Getting Started

### Prerequisites

- **Kubernetes 1.18 or higher version is required**

  You can use [kind](https://kind.sigs.k8s.io/) to create your own Kubernetes cluster:

  ```
  $ kind create cluster
  ```

### Quick start

Install your `gtctl` by one line:

```
$ curl -L https://raw.githubusercontent.com/greptimeteam/gtctl/develop/hack/install.sh | sh
```

After downloading, your `gtctl` will in the current directory.

If you want to install the specific version of `gtctl`, you can:

```
$ curl -L https://raw.githubusercontent.com/greptimeteam/gtctl/develop/hack/install.sh | sh -s <version>
```

Run `gtctl --hep` to get started:

```
$ gtctl --help
          __       __  __
   ____ _/ /______/ /_/ /
  / __ `/ __/ ___/ __/ /
 / /_/ / /_/ /__/ /_/ /
 \__, /\__/\___/\__/_/
/____/

gtctl is a command-line tool for managing GreptimeDB cluster.

Usage:
  gtctl [command]

Available Commands:
  cluster     Manage GreptimeDB cluster
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  version     Print the version of gtctl and exit

Flags:
  -h, --help      help for gtctl
  -v, --version   version for gtctl

Use "gtctl [command] --help" for more information about a command.
```

Create your own GreptimeDB cluster:

```
$ gtctl cluster create mydb -n default
```

After creating, the whole GreptimeDB cluster will start in `default` namespace:

```
# Get the cluster.
$ gtctl cluster get mydb -n default

# List all clusters.
$ gtctl cluster list
```

You can use `kubectl port-forward` command to forward frontend requests:

```
$ kubectl port-forward svc/mydb-frontend 3306:3306 > connections.out &
```

Use your `mysql` client to connect your cluster:

```
$ mysql -h 127.0.0.1 -P 3306
```

If you want to delete the cluster, you can:

```
# Just delete the cluster.
$ gtctl cluster delete mydb -n default

# Delete GreptimeDB cluster, including etcd cluster.
$ gtctl cluster delete mydb -n default --tear-down-etcd
```

### Dry Run Mode

gtctl provide `--dry-run` option in cluster creation. If the user execute the command with `--dry-run`, gtctl will output the manifests content without applying them:

```
$ gtctl cluster create mydb -n default --dry-run
```

### Experimental Feature

You can use the following commands to scale(or down-scale) your cluster:

```
# Scale datanode to 3 replicas.
$ gtctl cluster scale <your-cluster> -n <your-cluster-namespace> -c datanode --replicas 3

# Scale frontend to 5 replicas.
$ gtctl cluster scale <your-cluster> -n <your-cluster-namespace> -c frontend --replicas 5
```


## Development

- Compile the project

  ```
  $ make
  ```

  Then the `gtctl` will be generated in `bin/`.

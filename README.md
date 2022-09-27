# gtctl

## Overview

gtctl is a command-line tool for managing GreptimeDB cluster. gtctl integrates the multiple operations in one binary.

## Getting Started

### Prerequisites

- Kubernetes 1.18 or higher version is required

  You can use [kind](https://kind.sigs.k8s.io/) to create your own Kubernetes cluster:

  ```
  $ kind create cluster
  ```
### Usage

- Create GreptimeDB cluster

  ```
  $ gtctl create cluster -n mydb
  ```

- Get GreptimeDB cluster

  ```
  $ gtctl get cluster -n mydb
  ```

- Scale GreptimeDB cluster

  ```
  # Scale datanode to 3 replicas.
  $ gtctl scale cluster -n mydb -c datanode --replicas 3
  
  # Scale frontend to 5 replicas.
  $ gtctl scale cluster -n mydb -c frontend --replicas 5
  ```
  
- Delete GreptimeDB cluster

  ```
  # Delete GreptimeDB cluster.
  $ gtctl delete cluster -n mydb
  
  # Delete GreptimeDB cluster, including etcd cluster.
  $ gtctl delete cluster -n mydb --tear-down-etcd
  ```

## Development

Just make it:

```
$ make
```

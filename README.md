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

Install the `gtctl` executable with:

```shell
go install github.com/GreptimeTeam/gtctl/cmd/gtctl@develop
```

In case of `go` is _not_ available, try the following one-liner installation command:

```shell
curl -fsSL https://raw.githubusercontent.com/greptimeteam/gtctl/develop/hack/install.sh | sh
```

## Quickstart

The **fastest** way to experience the GreptimeDB cluster is to use the playground:

```shell
gtctl playground
```

The `playground` will deploy the minimal GreptimeDB cluster on your environment in bare-metal mode.

## Documentation

* [More](https://docs.greptime.com/user-guide/operations/gtctl) features and usage about `gtctl`

## License

`gtctl` uses the [Apache 2.0 license](LICENSE) to strike a balance between open contributions and allowing you to use the software however you want.

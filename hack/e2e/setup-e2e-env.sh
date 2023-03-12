#!/usr/bin/env bash
# Copyright 2023 Greptime Team
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


set -o errexit
set -o nounset
set -o pipefail

CLUSTER=e2e-cluster
REGISTRY_NAME=kind-registry
REGISTRY_PORT=5001

function check_prerequisites() {
    if ! hash docker 2>/dev/null; then
        echo "docker command is not found! You can download docker here: https://docs.docker.com/get-docker/"
        exit
    fi

    if ! hash kind 2>/dev/null; then
        echo "kind command is not found! You can download kind here: https://kind.sigs.k8s.io/docs/user/quick-start/#installing-from-release-binaries"
        exit
    fi

    if ! hash kubectl 2>/dev/null; then
        echo "kubectl command is not found! You can download kubectl here: https://kubernetes.io/docs/tasks/tools/"
        exit
    fi
}

function start_local_registry() {
    # create registry container unless it already exists
    if [ "$(docker inspect -f '{{.State.Running}}' "${REGISTRY_NAME}" 2>/dev/null || true)" != 'true' ]; then
        docker run \
        -d --restart=always -p "127.0.0.1:${REGISTRY_PORT}:5000" --name "${REGISTRY_NAME}" \
        registry:2
    fi
}

function create_kind_cluster() {
    # check cluster
    for cluster in $(kind get clusters); do
      if [ "$cluster" = "${CLUSTER}" ]; then
          echo "Use the existed cluster $cluster"
          kubectl config use-context kind-"$cluster"
          return
      fi
    done

    # create a cluster with the local registry enabled in containerd
    cat <<EOF | kind create cluster --name "${CLUSTER}" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:${REGISTRY_PORT}"]
    endpoint = ["http://${REGISTRY_NAME}:5000"]
nodes:
- role: control-plane
- role: worker
- role: worker
EOF

    # connect the registry to the cluster network if not already connected
    if [ "$(docker inspect -f='{{json .NetworkSettings.Networks.kind}}' "${REGISTRY_NAME}")" = 'null' ]; then
        docker network connect "kind" "${REGISTRY_NAME}"
    fi

    # Document the local registry
    # https://github.com/kubernetes/enhancements/tree/master/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:${REGISTRY_PORT}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF
}

check_prerequisites
start_local_registry
create_kind_cluster

#!/usr/bin/env bash

# 
# Runs a test of the knative serving installation by deploying and then 
# invoking an http echoing server.
#

set -o errexit
set -o nounset
set -o pipefail

TERM="${TERM:-dumb}"

main() {
  local em=$(tput bold)$(tput setaf 2)
  local me=$(tput sgr0)

  # Drop some debug in the event even the above excessive wait does not work.
  echo "${em}Testing...${me}"

  echo "${em}  Creating echo server${me}"
  n=0
  until [ $n -ge 10 ]; do
    cat <<EOF | kubectl apply -f - && break
  apiVersion: serving.knative.dev/v1
  kind: Service
  metadata:
    name: echo
    namespace: func
  spec:
    template:
      spec:
        containers:
          - image: docker.io/jmalloc/echo-server
EOF
    echo "Retrying..."
    sleep 5
  done

  sleep 60

  # wait for the route to become ready
  kubectl wait --for=condition=Ready route echo -n func

  echo "${em}  Invoking echo server${me}"
  curl http://echo.func.127.0.0.1.sslip.io/

  echo "${em}DONE${me}"

}

main "$@"



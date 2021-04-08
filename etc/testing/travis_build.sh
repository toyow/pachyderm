#!/bin/bash

set -ex

git config user.email "donotreply@pachyderm.com"
git config user.name "anonymous"

make install
version=$(pachctl version --client-only)
git tag -f -am "Circle CI test v${version}" v"${version}"
make docker-build
docker tag "pachyderm/pachd:local" "pachyderm/pachd:${version}"
docker push "pachyderm/pachd:${version}"
docker tag "pachyderm/worker:local" "pachyderm/worker:${version}"
docker push "pachyderm/worker:${version}"

# Push pipeline build images
make docker-build-pipeline-build
make docker-push-pipeline-build

#!/bin/bash

export SLUG=ghcr.io/awakari/subscriptions
export VERSION=latest
docker tag awakari/subscriptions "${SLUG}":"${VERSION}"
docker push "${SLUG}":"${VERSION}"

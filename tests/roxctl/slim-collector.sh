#! /usr/bin/env bash

set -uo pipefail

# This test script requires API_ENDPOINT and ROX_PASSWORD to be set in the environment.

[ -n "$API_ENDPOINT" ]
[ -n "$ROX_PASSWORD" ]

echo "Using API_ENDPOINT $API_ENDPOINT"

FAILURES=0

eecho() {
  echo "$@" >&2
}

die() {
    eecho "$@"
    exit 1
}

curl_central() {
  url="$1"
  shift
  [[ -n "${url}" ]] || die "No URL specified"
  curl -Sskf -u "admin:${ROX_PASSWORD}" "https://${API_ENDPOINT}/${url}" "$@"
}

# Retrieve API token
API_TOKEN_JSON="$(curl_central v1/apitokens/generate \
  -d '{"name": "test", "role": "Admin"}')" \
  || die "Failed to retrieve Rox API token"
ROX_API_TOKEN="$(echo "$API_TOKEN_JSON" | jq -er .token)" \
  || die "Failed to retrieve token from JSON"
export ROX_API_TOKEN

ENABLED="$(curl_central v1/featureflags | jq ".featureFlags[] | select(.envVar==\"ROX_SUPPORT_SLIM_COLLECTOR_MODE\") | .enabled")"
if [[ "${ENABLED}" != "true" ]]; then
    echo "Slim collector is not enabled in central. Skipping test."
    exit 0
fi

test_collector_image_references_in_deployment_bundles() {
    SLIM_COLLECTOR_FLAG="$1"
    EXPECTED_IMAGE_TAG="$2"

    CLUSTER_NAME="test-cluster-$RANDOM"
    echo "Testing correctness of collector image references for clusters generated with $SLIM_COLLECTOR_FLAG (cluster name is $CLUSTER_NAME)"

    # Verify that generating a cluster works.
    if OUTPUT="$(roxctl --insecure-skip-tls-verify --insecure -e "$API_ENDPOINT" \
    sensor generate k8s --name "$CLUSTER_NAME" "$SLIM_COLLECTOR_FLAG" 2>&1)"; then
        echo "[OK] Generating cluster works"
    else
        eecho "[FAIL] Failed to generate cluster"
        eecho "Captured output was:"
        eecho "$OUTPUT"
        printf "\n\n" >&2
        FAILURES=$((FAILURES + 1))
        return
    fi

    cluster_id="$(curl_central v1/clusters | jq --arg name "${CLUSTER_NAME}" '.clusters | .[] | select(.name==$name).id' -r)"
    if [[ -n "${cluster_id}" ]]; then
        echo "[OK] Got cluster id ${cluster_id}"
    else
        eecho "[FAIL] Failed to retrieve cluster id"
        FAILURES=$((FAILURES + 1))
        return
    fi

    # Verify that generated bundle references the expected collector image.
    COLLECTOR_IMAGE_TAG="$(egrep 'image: \S+/collector' "sensor-${CLUSTER_NAME}/collector.yaml" | sed -e 's/[^:]*: "[^:]*:\(.*\)"$/\1/;')"
    COLLECTOR_IMAGE_TAG_SUFFIX="$(echo "$COLLECTOR_IMAGE_TAG" | sed -e 's/.*-\([^-]*\)$/\1/;')"

    if [ "$COLLECTOR_IMAGE_TAG_SUFFIX" == "$EXPECTED_IMAGE_TAG" ]; then
        echo "[OK] Newly generated bundle references $EXPECTED_IMAGE_TAG collector image"
    else
        eecho "[FAIL] Newly generated bundle does not reference $EXPECTED_IMAGE_TAG collector image (referenced collector image tag is $COLLECTOR_IMAGE_TAG)"
        FAILURES=$((FAILURES + 1))
        return
    fi

    rm -r "sensor-${CLUSTER_NAME}"

    # Verify that refetching deployment bundle for newly created cluster works as expected (i.e. that the bundle references the expected collector image).
    OUTPUT="$(roxctl --insecure-skip-tls-verify --insecure -e "$API_ENDPOINT" \
    sensor get-bundle --output-dir="sensor-${CLUSTER_NAME}-refetched" "$CLUSTER_NAME" 2>&1)"
    COLLECTOR_IMAGE_TAG="$(egrep 'image: \S+/collector' "sensor-${CLUSTER_NAME}-refetched/collector.yaml" | sed -e 's/[^:]*: "[^:]*:\(.*\)"$/\1/;')"
    COLLECTOR_IMAGE_TAG_SUFFIX="$(echo "$COLLECTOR_IMAGE_TAG" | sed -e 's/.*-\([^-]*\)$/\1/;')"

    if [ "$COLLECTOR_IMAGE_TAG_SUFFIX" == "$EXPECTED_IMAGE_TAG" ]; then
        echo "[OK] Refetched deployment bundle still references $EXPECTED_IMAGE_TAG collector image"
    else
        eecho "[FAIL] Refetched deployment bundle does not reference $EXPECTED_IMAGE_TAG collector image (referenced collector image tag is $COLLECTOR_IMAGE_TAG)"
        eecho "Captured output was:"
        eecho "$OUTPUT"
        FAILURES=$((FAILURES + 1))
        return
    fi

    rm -r "sensor-${CLUSTER_NAME}-refetched"

    curl_central "v1/clusters/${cluster_id}" -X DELETE
    if [[ $? -eq 0 ]]; then
        echo "[OK] Successfully cleaned up cluster"
    else
        eecho "[FAIL] Failed to delete cluster"
        FAILURES=$((FAILURES + 1))
        return
    fi
}

test_collector_image_references_in_deployment_bundles "--slim-collector" "slim"
test_collector_image_references_in_deployment_bundles "--slim-collector=auto" "slim" # Central is deployed in online mode in CI
test_collector_image_references_in_deployment_bundles "--slim-collector=false" "latest"

if [ $FAILURES -eq 0 ]; then
  echo "Passed"
else
  echo "$FAILURES tests failed"
  exit 1
fi

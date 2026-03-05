#!/usr/bin/env bash
# Build the test-runner Docker image, run block recording tests inside it,
# and copy the resulting test blocks to the host.
#
# Usage:
#   ./run_docker_recording_tests.sh [OPTIONS]
#
# Options:
#   --force-build   Rebuild Docker image even if it exists
#   --skip-build    Skip Docker image build
#   --image NAME    Docker image name (default: nitro-test-runner)

set -euo pipefail

IMAGE_NAME="${IMAGE_NAME:-nitro-test-runner}"
CONTAINER_NAME="nitro-test-runner-$$"
HOST_OUTPUT="system_tests/target"
CONTAINER_OUTPUT="/workspace/system_tests/target"

FORCE_BUILD=false
SKIP_BUILD=false

while [[ $# -gt 0 ]]; do
  case $1 in
    --force-build) FORCE_BUILD=true; shift ;;
    --skip-build)  SKIP_BUILD=true;  shift ;;
    --image)       IMAGE_NAME="$2";  shift 2 ;;
    -h|--help)
      echo "Usage: $0 [--force-build] [--skip-build] [--image NAME]"
      echo ""
      echo "  --force-build   Rebuild Docker image even if it exists"
      echo "  --skip-build    Skip Docker image build entirely"
      echo "  --image NAME    Docker image name (default: nitro-test-runner)"
      exit 0
      ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

# Clean up container on exit (if it was created)
CONTAINER_CREATED=false
cleanup() {
  if [ "$CONTAINER_CREATED" = true ]; then
    echo "Removing container '${CONTAINER_NAME}'..."
    docker rm "${CONTAINER_NAME}" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT
trap 'docker rm -f "${CONTAINER_NAME}" 2>/dev/null; exit 1' INT TERM

# ── Build ──────────────────────────────────────────────────────────────────
if [ "$SKIP_BUILD" = false ]; then
  if [ "$FORCE_BUILD" = true ] || ! docker image inspect "$IMAGE_NAME" &>/dev/null; then
    echo "Building Docker image '${IMAGE_NAME}' from Dockerfile.test-runner..."
    docker build -f Dockerfile.test-runner --target test-runner -t "${IMAGE_NAME}" .
  else
    echo "Using existing image '${IMAGE_NAME}' (use --force-build to rebuild)"
  fi
fi

# ── Run tests ──────────────────────────────────────────────────────────────
echo ""
echo "Running block recording tests in Docker..."
TEST_EXIT=0
docker run --name "${CONTAINER_NAME}" "${IMAGE_NAME}" || TEST_EXIT=$?
CONTAINER_CREATED=true

if [ "$TEST_EXIT" -ne 0 ]; then
  echo ""
  echo "WARNING: Tests exited with code ${TEST_EXIT}. Copying available results."
fi

# ── Copy results ───────────────────────────────────────────────────────────
echo ""
echo "Copying test results to ${HOST_OUTPUT}/..."
mkdir -p "${HOST_OUTPUT}"

if docker cp "${CONTAINER_NAME}:${CONTAINER_OUTPUT}/." "${HOST_OUTPUT}/" 2>/dev/null; then
  echo ""
  echo "Test blocks copied to ${HOST_OUTPUT}/:"
  COUNT=0
  for d in "${HOST_OUTPUT}"/TestRecord*; do
    if [ -d "$d" ]; then
      COUNT=$((COUNT + 1))
      echo "  $(basename "$d")"
    fi
  done
  echo ""
  echo "Total: ${COUNT} test block directories"
else
  echo "No test output found in container (tests may have all failed before producing output)."
fi

# ── Copy machine files ────────────────────────────────────────────────────
MACHINES_HOST="target/machines/latest"
MACHINES_CONTAINER="/workspace/target/machines/latest"

echo ""
echo "Copying machine files to ${MACHINES_HOST}/..."
mkdir -p "${MACHINES_HOST}"

if docker cp "${CONTAINER_NAME}:${MACHINES_CONTAINER}/." "${MACHINES_HOST}/" 2>/dev/null; then
  echo "Machine files copied to ${MACHINES_HOST}/:"
  for f in "${MACHINES_HOST}"/*; do
    if [ -f "$f" ]; then
      echo "  $(basename "$f")"
    fi
  done
else
  echo "No machine files found in container."
fi

exit "$TEST_EXIT"

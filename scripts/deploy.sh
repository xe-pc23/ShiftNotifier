#!/usr/bin/env bash
set -euo pipefail

APP_DIR="${SHIFT_NOTIFIER_APP_DIR:-$(pwd)}"
BRANCH="${SHIFT_NOTIFIER_DEPLOY_BRANCH:-main}"
BIN_PATH="${SHIFT_NOTIFIER_BIN_PATH:-shift-notifier}"
RUN_TESTS="${SHIFT_NOTIFIER_DEPLOY_RUN_TESTS:-true}"
SERVICE_NAME="${SHIFT_NOTIFIER_SERVICE_NAME:-shift-notifier}"
SYSTEMCTL="${SHIFT_NOTIFIER_SYSTEMCTL:-systemctl}"
RESTART_CMD="${SHIFT_NOTIFIER_RESTART_CMD:-}"
HEALTHCHECK_URL="${SHIFT_NOTIFIER_HEALTHCHECK_URL:-}"

cd "$APP_DIR"

if [ -n "$(git status --porcelain --untracked-files=no)" ]; then
  echo "working tree has uncommitted tracked changes; aborting deploy" >&2
  git status --short --untracked-files=no >&2
  exit 1
fi

git fetch origin "$BRANCH"
git checkout "$BRANCH"
git pull --ff-only origin "$BRANCH"

if [ "$RUN_TESTS" = "true" ]; then
  go test ./...
fi

BIN_DIR="$(dirname "$BIN_PATH")"
BIN_NAME="$(basename "$BIN_PATH")"
TEMP_BIN="$(mktemp "$BIN_DIR/.${BIN_NAME}.tmp.XXXXXX")"
trap 'rm -f "$TEMP_BIN"' EXIT

go build -o "$TEMP_BIN" ./cmd/shift-notifier
chmod 755 "$TEMP_BIN"
mv "$TEMP_BIN" "$BIN_PATH"
trap - EXIT

if [ -n "$RESTART_CMD" ]; then
  bash -lc "$RESTART_CMD"
elif [ "$(id -u)" = "0" ]; then
  "$SYSTEMCTL" restart "$SERVICE_NAME"
  "$SYSTEMCTL" is-active --quiet "$SERVICE_NAME"
else
  sudo "$SYSTEMCTL" restart "$SERVICE_NAME"
  sudo "$SYSTEMCTL" is-active --quiet "$SERVICE_NAME"
fi

if [ -n "$HEALTHCHECK_URL" ]; then
  curl --fail --silent --show-error --max-time 10 "$HEALTHCHECK_URL" >/dev/null
fi

echo "deploy completed"

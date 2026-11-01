#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

: "${UMAMI_URL:=http://localhost:3999}"
: "${UMAMI_USERNAME:=admin}"
: "${UMAMI_PASSWORD:=umami}"
export UMAMI_URL UMAMI_USERNAME UMAMI_PASSWORD
export PULUMI_CONFIG_PASSPHRASE=""

state="$(mktemp -d)"
export PULUMI_BACKEND_URL="file://${state}"
trap 'pulumi destroy --yes --non-interactive >/dev/null 2>&1 || true; pulumi stack rm e2e --yes --force >/dev/null 2>&1 || true; rm -rf "${state}"' EXIT

api() {
  curl -sf -H "Authorization: Bearer ${token}" -H 'Content-Type: application/json' "$@"
}

step() {
  printf '\n==> %s\n' "$1"
}

token="$(curl -sf -X POST "${UMAMI_URL}/api/auth/login" -H 'Content-Type: application/json' \
  -d "{\"username\":\"${UMAMI_USERNAME}\",\"password\":\"${UMAMI_PASSWORD}\"}" | jq -r .token)"

step "install"
bun install --silent
pulumi stack init e2e --non-interactive

step "preview and up"
pulumi preview --non-interactive --expect-no-changes && exit 1 || true
pulumi up --yes --non-interactive
id="$(pulumi stack output websiteId)"
api "${UMAMI_URL}/api/websites/${id}" | jq -e '.name == "Example" and .domain == "example.com"' >/dev/null

step "refresh without drift"
pulumi refresh --yes --non-interactive --expect-no-changes

step "update"
pulumi config set siteName Renamed
pulumi up --yes --non-interactive
api "${UMAMI_URL}/api/websites/${id}" | jq -e '.name == "Renamed"' >/dev/null

step "drift detection"
api -X POST "${UMAMI_URL}/api/websites/${id}" -d '{"name":"Drifted","domain":"drift.example.com"}' >/dev/null
pulumi refresh --yes --non-interactive --expect-no-changes && exit 1 || true
pulumi up --yes --non-interactive
api "${UMAMI_URL}/api/websites/${id}" | jq -e '.name == "Renamed" and .domain == "example.com"' >/dev/null

step "import"
imported="$(api -X POST "${UMAMI_URL}/api/websites" -d '{"name":"Imported","domain":"imported.example.com"}' | jq -r .id)"
pulumi import umami:index:Website imported "${imported}" --yes --non-interactive --skip-preview --protect=false >/dev/null
pulumi stack export | jq -e --arg id "${imported}" '.deployment.resources[] | select(.id == $id)' >/dev/null

step "out of band deletion"
api -X DELETE "${UMAMI_URL}/api/websites/${id}" >/dev/null
pulumi refresh --yes --non-interactive --expect-no-changes && exit 1 || true
pulumi up --yes --non-interactive
new_id="$(pulumi stack output websiteId)"
test "${new_id}" != "${id}"

step "destroy"
pulumi destroy --yes --non-interactive
api "${UMAMI_URL}/api/websites/${new_id}" | jq -e '. == null or .id == null' >/dev/null
api "${UMAMI_URL}/api/websites/${imported}" | jq -e '. == null or .id == null' >/dev/null

printf '\nAll end-to-end checks passed.\n'

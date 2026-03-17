#!/usr/bin/env bash
# pre-pr-check.sh — run all CI checks locally before pushing a PR.
# Mirrors: quality-and-security.yml, secret_scanning.yml (gitleaks).
# CodeQL (GitHub-only) and SBOM (artifact-only) are intentionally skipped.
#
# Dependency graph (matches CI jobs):
#   Wave 1 (parallel): gofmt | go_vet | go_test | conftest | gitleaks
#   Wave 2:            govulncheck    <- needs gofmt + go_vet + go_test
#   Wave 3:            go_build       <- needs govulncheck
#   conftest / gitleaks run freely and never block other waves.
set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

# ── colors ────────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; DIM='\033[2m'; RESET='\033[0m'

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

declare -A PIDS STATUS
declare -a ORDER PENDING

# ── helpers ───────────────────────────────────────────────────────────────────
banner() { echo -e "\n${CYAN}${BOLD}── $* ──${RESET}"; }
has_tool() { command -v "$1" &>/dev/null; }

# ── hints (shown on failure) ──────────────────────────────────────────────────
declare -A HINTS
HINTS[gofmt]="run: gofmt -w . to auto-fix all formatting"
HINTS[go_vet]="fix the reported issues; run: go vet ./..."
HINTS[go_test]="fix failing tests; run: make test-distributed (or: go test ./... -v -run <TestName>)"
HINTS[govulncheck]="upgrade the vulnerable module: go get -u <module>@<safe-version>"
HINTS[conftest]="fix the Dockerfile policy violations listed above"
HINTS[gitleaks]="remove the secret from history; see: git-filter-repo or BFG"
HINTS[go_build]="fix compilation errors above; run: go build ./... for details"

# ── launch ────────────────────────────────────────────────────────────────────
# Run a check in the background. Output is buffered to a log file; the exit
# code is written to a status file when the command finishes. The section is
# printed by flush_completed() once that status file appears.
launch() {
    local name="$1"; shift
    ORDER+=("$name")
    PENDING+=("$name")
    local log="${WORKDIR}/${name}.log"
    local sf="${WORKDIR}/${name}.status"
    (
        "$@" > "$log" 2>&1
        echo $? > "$sf"
    ) &
    PIDS["$name"]=$!
    printf "${DIM}  %-14s started${RESET}\n" "$name"
}

# ── mark_skip ─────────────────────────────────────────────────────────────────
mark_skip() {
    local name="$1" reason="$2"
    ORDER+=("$name")
    STATUS["$name"]=skip
    echo -e "\n${YELLOW}${BOLD}┌─ $name${RESET}"
    echo -e "${YELLOW}│${RESET}  skipped — $reason"
    echo -e "${YELLOW}${BOLD}└─ SKIP${RESET}"
}

# ── print_section ─────────────────────────────────────────────────────────────
print_section() {
    local name="$1"
    local code="${STATUS[$name]}"
    local log="${WORKDIR}/${name}.log"
    local color label
    if [[ "$code" -eq 0 ]]; then color="$GREEN"; label="PASS"
    else                          color="$RED";   label="FAIL"
    fi

    echo -e "\n${color}${BOLD}┌─ $name${RESET}"
    if [[ -f "$log" && -s "$log" ]]; then
        while IFS= read -r line; do
            printf "${color}│${RESET}  %s\n" "$line"
        done < "$log"
    fi
    echo -e "${color}${BOLD}└─ $label${RESET}"
    if [[ "$code" -ne 0 && -n "${HINTS[$name]:-}" ]]; then
        echo -e "   ${DIM}hint: ${HINTS[$name]}${RESET}"
    fi
}

# ── flush_completed ───────────────────────────────────────────────────────────
# Print sections for any PENDING checks whose status file has appeared.
# Removes printed checks from PENDING.
flush_completed() {
    local still_pending=()
    for name in "${PENDING[@]+"${PENDING[@]}"}"; do
        local sf="${WORKDIR}/${name}.status"
        if [[ -f "$sf" ]]; then
            wait "${PIDS[$name]}" 2>/dev/null || true
            STATUS["$name"]=$(cat "$sf")
            print_section "$name"
        else
            still_pending+=("$name")
        fi
    done
    PENDING=("${still_pending[@]+"${still_pending[@]}"}")
}

# ── wait_for ──────────────────────────────────────────────────────────────────
# Block until all named checks have statuses, flushing completed sections
# (including any other in-flight checks) along the way.
wait_for() {
    while true; do
        flush_completed
        local all_done=true
        for name in "$@"; do
            [[ "${STATUS[$name]+_}" ]] || { all_done=false; break; }
        done
        $all_done && break
        sleep 0.1
    done
}

# ── gate_ok ───────────────────────────────────────────────────────────────────
# Returns 0 only if all named checks have exit code 0.
gate_ok() {
    for name in "$@"; do
        local s="${STATUS[$name]:-1}"
        [[ "$s" =~ ^[0-9]+$ && "$s" -eq 0 ]] || return 1
    done
}

# ─────────────────────────────────────────────────────────────────────────────
# Wave 1 — all independent checks in parallel
# ─────────────────────────────────────────────────────────────────────────────
banner "Wave 1 — parallel: gofmt | go_vet | go_test | conftest | gitleaks"

launch gofmt bash -c '
    files="$(gofmt -l .)"
    if [ -n "$files" ]; then
        echo "Files needing formatting:"
        printf "  %s\n" $files
        exit 1
    fi
    echo "all files properly formatted"
'

launch go_vet go vet ./...

if has_tool docker; then
    launch go_test bash -c '
        REDIS_NAME="wacraft-pre-pr-redis-$$"
        PG_NAME="wacraft-pre-pr-postgres-$$"
        cleanup() { docker rm -f "$REDIS_NAME" "$PG_NAME" >/dev/null 2>&1 || true; }
        trap cleanup EXIT

        echo "Starting ephemeral Redis..."
        docker run -d --name "$REDIS_NAME" -p 0:6379 redis:7-alpine >/dev/null
        REDIS_PORT=$(docker inspect --format="{{(index (index .NetworkSettings.Ports \"6379/tcp\") 0).HostPort}}" "$REDIS_NAME")

        echo "Starting ephemeral PostgreSQL..."
        docker run -d --name "$PG_NAME" -p 0:5432 \
            -e POSTGRES_DB=postgres \
            -e POSTGRES_USER=postgres \
            -e POSTGRES_PASSWORD=postgres \
            postgres:17-alpine >/dev/null
        PG_PORT=$(docker inspect --format="{{(index (index .NetworkSettings.Ports \"5432/tcp\") 0).HostPort}}" "$PG_NAME")

        echo "Waiting for Redis (port $REDIS_PORT)..."
        until docker exec "$REDIS_NAME" redis-cli ping 2>/dev/null | grep -q PONG; do sleep 0.1; done

        echo "Waiting for PostgreSQL (port $PG_PORT)..."
        until docker exec "$PG_NAME" pg_isready -U postgres -q 2>/dev/null; do sleep 0.1; done

        echo "Running tests..."
        REDIS_URL="redis://localhost:$REDIS_PORT" \
        DATABASE_URL="postgres://postgres:postgres@localhost:$PG_PORT/postgres?sslmode=disable" \
        go test ./... -v -race -count=1
    '
else
    echo -e "  ${YELLOW}docker not found — running tests without DB/Redis (integration tests will be skipped)${RESET}"
    launch go_test go test ./... -v -count=1
fi

if has_tool conftest; then
    launch conftest conftest test Dockerfile --policy policy/
else
    mark_skip conftest "not installed — https://www.conftest.dev/install/"
fi

if has_tool gitleaks; then
    launch gitleaks gitleaks detect --source . -v
else
    mark_skip gitleaks "not installed — https://github.com/gitleaks/gitleaks#installing"
fi

# ─────────────────────────────────────────────────────────────────────────────
# Gate: wait for lint checks before proceeding to govulncheck.
# conftest/gitleaks sections also appear here as they complete.
# ─────────────────────────────────────────────────────────────────────────────
banner "Gate — gofmt | go_vet | go_test"
wait_for gofmt go_vet go_test

# ─────────────────────────────────────────────────────────────────────────────
# Wave 2 — govulncheck (needs lint gate)
# ─────────────────────────────────────────────────────────────────────────────
if gate_ok gofmt go_vet go_test; then
    banner "Wave 2 — govulncheck"
    GOVULNCHECK="$(go env GOPATH)/bin/govulncheck"
    if [[ ! -x "$GOVULNCHECK" ]]; then
        echo "  Installing govulncheck..."
        go install golang.org/x/vuln/cmd/govulncheck@latest
    fi
    launch govulncheck "$GOVULNCHECK" ./...

    banner "Gate — govulncheck"
    wait_for govulncheck

    # ── Wave 3 — go build (needs govulncheck) ─────────────────────────────
    if gate_ok govulncheck; then
        banner "Wave 3 — go_build"
        mkdir -p ./bin
        launch go_build go build -o ./bin/wacraft-server
        wait_for go_build
    else
        mark_skip go_build "govulncheck failed"
    fi
else
    mark_skip govulncheck "lint gate failed"
    mark_skip go_build    "lint gate failed"
fi

# ─────────────────────────────────────────────────────────────────────────────
# Drain any still-running wave-1 checks (conftest, gitleaks)
# ─────────────────────────────────────────────────────────────────────────────
if [[ ${#PENDING[@]} -gt 0 ]]; then
    banner "Waiting for remaining checks…"
    while [[ ${#PENDING[@]} -gt 0 ]]; do
        flush_completed
        [[ ${#PENDING[@]} -gt 0 ]] && sleep 0.1
    done
fi

# ─────────────────────────────────────────────────────────────────────────────
# Summary
# ─────────────────────────────────────────────────────────────────────────────
echo -e "\n${BOLD}══════════════════════════════════════════${RESET}"
echo -e "${BOLD} Pre-PR Check Summary${RESET}"
echo -e "${BOLD}══════════════════════════════════════════${RESET}\n"

any_failed=false
for name in "${ORDER[@]}"; do
    s="${STATUS[$name]:-?}"
    if   [[ "$s" == skip ]]; then
        printf "  ${YELLOW}SKIP${RESET}  %s\n" "$name"
    elif [[ "$s" =~ ^[0-9]+$ && "$s" -eq 0 ]]; then
        printf "  ${GREEN}PASS${RESET}  %s\n" "$name"
    else
        printf "  ${RED}FAIL${RESET}  %s\n" "$name"
        printf "        ${DIM}hint: %s${RESET}\n" "${HINTS[$name]:-see output above}"
        any_failed=true
    fi
done

echo ""
if $any_failed; then
    echo -e "${RED}${BOLD}Fix the issues above before opening a PR.${RESET}"
    exit 1
else
    echo -e "${GREEN}${BOLD}All checks passed. Safe to open a PR.${RESET}"
    exit 0
fi

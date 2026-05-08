#!/usr/bin/env bash
# prepare-release.sh — bump the API version and regenerate Swagger docs
# in preparation for a release. Run locally before tagging/publishing.
#
# Steps:
#   1. Fetch the latest GitHub release tag for this repo.
#   2. Compute the next patch version (X.Y.Z -> X.Y.(Z+1)).
#   3. Rewrite the `// @version` line in main.go to the new version.
#   4. Regenerate Swagger docs via `swag init --parseDependency`.
#
# Requires: git, swag, and either `gh` (preferred) or `curl`+`jq`.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

# ── colors ────────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; DIM='\033[2m'; RESET='\033[0m'

info()  { echo -e "${CYAN}${BOLD}==>${RESET} $*"; }
warn()  { echo -e "${YELLOW}${BOLD}!! ${RESET}$*"; }
ok()    { echo -e "${GREEN}${BOLD}✓${RESET} $*"; }
fail()  { echo -e "${RED}${BOLD}✗${RESET} $*" >&2; exit 1; }

has_tool() { command -v "$1" &>/dev/null; }

# ── locate target repo ────────────────────────────────────────────────────────
# Parse the `origin` remote (git@github.com:OWNER/REPO.git or https URL).
detect_repo() {
    local url owner_repo
    url="$(git remote get-url origin 2>/dev/null || true)"
    [[ -n "$url" ]] || fail "no 'origin' git remote found"

    # Strip prefixes/suffixes to leave OWNER/REPO.
    owner_repo="${url#git@github.com:}"
    owner_repo="${owner_repo#https://github.com/}"
    owner_repo="${owner_repo#http://github.com/}"
    owner_repo="${owner_repo%.git}"

    [[ "$owner_repo" == */* ]] || fail "could not parse OWNER/REPO from remote: $url"
    echo "$owner_repo"
}

REPO="$(detect_repo)"
info "Repository: ${BOLD}$REPO${RESET}"

# ── fetch latest release tag ──────────────────────────────────────────────────
fetch_latest_tag() {
    local tag=""
    if has_tool gh; then
        tag="$(gh release view --repo "$REPO" --json tagName -q .tagName 2>/dev/null || true)"
    fi
    if [[ -z "$tag" ]]; then
        if ! has_tool curl; then
            fail "neither 'gh' nor 'curl' available — cannot query GitHub"
        fi
        local api="https://api.github.com/repos/$REPO/releases/latest"
        local hdrs=(-H "Accept: application/vnd.github+json")
        [[ -n "${GITHUB_TOKEN:-}" ]] && hdrs+=(-H "Authorization: Bearer $GITHUB_TOKEN")
        local body
        body="$(curl -fsSL "${hdrs[@]}" "$api" 2>/dev/null || true)"
        [[ -n "$body" ]] || fail "failed to fetch latest release from $api"
        if has_tool jq; then
            tag="$(echo "$body" | jq -r '.tag_name // empty')"
        else
            # crude fallback: grep the tag_name field
            tag="$(echo "$body" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
        fi
    fi
    [[ -n "$tag" ]] || fail "could not determine latest release tag"
    echo "$tag"
}

info "Fetching latest GitHub release..."
LATEST_TAG="$(fetch_latest_tag)"
ok "Latest release: ${BOLD}$LATEST_TAG${RESET}"

# Strip optional leading 'v'.
LATEST_VERSION="${LATEST_TAG#v}"

# Validate semver (X.Y.Z, optionally with pre-release/build that we discard).
if [[ ! "$LATEST_VERSION" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+) ]]; then
    fail "latest tag '$LATEST_TAG' is not a valid X.Y.Z version"
fi
MAJOR="${BASH_REMATCH[1]}"
MINOR="${BASH_REMATCH[2]}"
PATCH="${BASH_REMATCH[3]}"
NEW_PATCH=$((PATCH + 1))
NEW_VERSION="${MAJOR}.${MINOR}.${NEW_PATCH}"
ok "Next patch version: ${BOLD}$NEW_VERSION${RESET}"

# ── update main.go ────────────────────────────────────────────────────────────
MAIN_GO="$REPO_ROOT/main.go"
[[ -f "$MAIN_GO" ]] || fail "main.go not found at $MAIN_GO"

# Capture the current version for reporting/no-op detection.
CURRENT_VERSION="$(sed -n 's|^// @version[[:space:]]\+\([0-9][0-9.]*\).*|\1|p' "$MAIN_GO" | head -n1)"
[[ -n "$CURRENT_VERSION" ]] || fail "could not find '// @version' line in main.go"
info "Current main.go version: ${BOLD}$CURRENT_VERSION${RESET}"

if [[ "$CURRENT_VERSION" == "$NEW_VERSION" ]]; then
    warn "main.go already at $NEW_VERSION — skipping rewrite"
else
    # Replace only the version token, preserving the original whitespace
    # (tabs) between `@version` and the value.
    if ! sed -i.bak -E "s|^(// @version[[:space:]]+)[0-9]+\.[0-9]+\.[0-9]+([^[:space:]]*)|\1${NEW_VERSION}|" "$MAIN_GO"; then
        fail "failed to update version in main.go"
    fi
    rm -f "$MAIN_GO.bak"
    # Verify the change landed.
    UPDATED_VERSION="$(sed -n 's|^// @version[[:space:]]\+\([0-9][0-9.]*\).*|\1|p' "$MAIN_GO" | head -n1)"
    [[ "$UPDATED_VERSION" == "$NEW_VERSION" ]] || fail "main.go version did not update (still $UPDATED_VERSION)"
    ok "Updated main.go: $CURRENT_VERSION -> $NEW_VERSION"
fi

# ── regenerate Swagger docs ───────────────────────────────────────────────────
if ! has_tool swag; then
    fail "'swag' not installed — run: go install github.com/swaggo/swag/cmd/swag@latest"
fi

info "Regenerating Swagger docs (swag init --parseDependency)..."
swag init --parseDependency
ok "Swagger docs regenerated"

# ── summary ───────────────────────────────────────────────────────────────────
echo ""
echo -e "${GREEN}${BOLD}══════════════════════════════════════════${RESET}"
echo -e "${GREEN}${BOLD} Release prep complete${RESET}"
echo -e "${GREEN}${BOLD}══════════════════════════════════════════${RESET}"
echo -e "  Repository:      ${BOLD}$REPO${RESET}"
echo -e "  Latest release:  ${BOLD}$LATEST_TAG${RESET}"
echo -e "  New API version: ${BOLD}$NEW_VERSION${RESET}"
echo ""
echo -e "${DIM}Review the diff (main.go, docs/) and commit before tagging.${RESET}"

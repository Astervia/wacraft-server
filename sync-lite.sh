#!/usr/bin/env bash
set -euo pipefail

# === CONFIGURATION ===
FULL_REPO="git@github.com:Astervia/wacraft-server.git"
LITE_REPO="git@github.com:Astervia/wacraft-server-lite.git"
TMP_DIR="temp-lite"

# List of files to delete (relative paths)
FILES_TO_DELETE=(
    ".github/workflows/codeql-analysis.yml" # Delete to set with upload: true
    ".github/workflows/release.yml" # Delete to set with correct image name
)

# List of folders to delete (relative paths)
FOLDERS_TO_DELETE=(
    "src/campaign"
)

# === CLEAN UP OLD TEMP DIR ===
rm -rf "$TMP_DIR"

# === CLONE FULL REPO ===
git clone "$FULL_REPO" "$TMP_DIR"
cd "$TMP_DIR"

# === CREATE NEW BRANCH BASED ON MAIN ===
git checkout -b prepare-lite origin/main

# === REMOVE // PREMIUM STARTS ... // PREMIUM ENDS BLOCKS IN ALL FILES ===
FILES=$(git ls-files)

for file in $FILES; do
  if [[ -f "$file" ]]; then
    perl -0777 -i -pe 's{^[ \t]*// PREMIUM STARTS.*?// PREMIUM ENDS\s*}{}gms' "$file"
  fi
done

# === DELETE SPECIFIC FILES ===
for f in "${FILES_TO_DELETE[@]}"; do
  if [[ -f "$f" ]]; then
    git rm "$f"
  fi
done

# === DELETE SPECIFIC FOLDERS ===
for d in "${FOLDERS_TO_DELETE[@]}"; do
  if [[ -d "$d" ]]; then
    git rm -r "$d"
  fi
done

# === RUN GOFMT ON ALL GO FILES ===
echo "Running gofmt..."
GO_FILES=$(find . -name "*.go" -type f)

if [ -n "$GO_FILES" ]; then
  gofmt -s -w $GO_FILES
fi

# === REGENERATE FILES ===
make build

# === ADD & COMMIT CHANGES ===
git add -u
git commit -m "Prepare lite version: remove premium content and regenerate docs"

# === PUSH TO LITE REPO ===
git remote set-url origin "$LITE_REPO"
git push --force origin prepare-lite:main

echo "✅ wacraft-server-lite updated successfully!"

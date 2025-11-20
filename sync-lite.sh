#!/usr/bin/env bash
set -euo pipefail

# === CONFIGURATION ===
FULL_REPO="git@github.com:Astervia/wacraft-server.git"
LITE_REPO="git@github.com:Astervia/wacraft-server-lite.git"
TMP_DIR="temp-lite"
LITE_REMOTE_NAME="lite"
LITE_IGNORE_FILE=".liteignore"

# List of files to delete (relative paths in FULL repo)
FILES_TO_DELETE=(
    ".github/workflows/codeql-analysis.yml" # Delete to set with upload: true
    ".github/workflows/release.yml"         # Delete to set with correct image name
)

# List of folders to delete (relative paths)
FOLDERS_TO_DELETE=(
    "src/campaign"
)

# === HELPER: load .liteignore ===
LITE_PROTECTED=()

if [[ -f "$LITE_IGNORE_FILE" ]]; then
  echo "Loading protected paths from $LITE_IGNORE_FILE..."
  while IFS= read -r line || [[ -n "$line" ]]; do
    # Skip empty lines and comments
    [[ -z "$line" || "$line" == \#* ]] && continue
    LITE_PROTECTED+=("$line")
  done < "$LITE_IGNORE_FILE"
else
  echo "No $LITE_IGNORE_FILE found. No protected lite-only files will be restored."
fi

restore_protected_from_lite() {
  if [[ ${#LITE_PROTECTED[@]} -eq 0 ]]; then
    echo "No protected paths to restore from lite repo."
    return 0
  fi

  echo "Adding lite remote and fetching main..."
  git remote add "$LITE_REMOTE_NAME" "$LITE_REPO"
  git fetch "$LITE_REMOTE_NAME" main

  echo "Restoring protected paths from $LITE_REMOTE_NAME/main..."
  for path in "${LITE_PROTECTED[@]}"; do
    # Check if path exists in lite/main before attempting checkout
    if git cat-file -e "${LITE_REMOTE_NAME}/main:${path}" 2>/dev/null; then
      echo "  - Restoring $path from lite/main"
      git checkout "${LITE_REMOTE_NAME}/main" -- "$path"
    else
      echo "  - ⚠️  Protected path '$path' not found in lite/main, skipping."
    fi
  done
}

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
    echo "Removing file: $f"
    git rm "$f"
  fi
done

# === DELETE SPECIFIC FOLDERS ===
for d in "${FOLDERS_TO_DELETE[@]}"; do
  if [[ -d "$d" ]]; then
    echo "Removing folder: $d"
    git rm -r "$d"
  fi
done

# === RUN GOFMT ON ALL GO FILES ===
echo "Running gofmt..."
GO_FILES=$(find . -name "*.go" -type f)

if [[ -n "$GO_FILES" ]]; then
  gofmt -s -w $GO_FILES
fi

# === REGENERATE FILES ===
echo "Running make build..."
make build

# === RESTORE LITE-ONLY FILES (e.g., codeql-analysis-lite.yml, release-lite.yml) ===
restore_protected_from_lite

# === ADD & COMMIT CHANGES ===
git add -u
git commit -m "Prepare lite version: remove premium content and regenerate docs"

# === PUSH TO LITE REPO (WITHOUT TOUCHING FULL ORIGIN) ===
echo "Pushing to lite repo main (force)..."
git push --force "$LITE_REPO" HEAD:main

echo "✅ wacraft-server-lite updated successfully!"

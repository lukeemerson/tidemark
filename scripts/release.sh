#!/bin/sh
# release.sh v0.1.0 — test, build the darwin/arm64 package, and create a DRAFT GitHub release with it.
# Publishing is left to you: press "Publish release" on the draft (or: gh release edit <tag> --draft=false).
set -eu
tag=${1:?usage: scripts/release.sh v0.1.0}
cd "$(dirname "$0")/.."
name=tidemark_darwin_arm64

go test ./...
rm -rf dist
mkdir -p "dist/$name"
GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o "dist/$name/tidemark" ./cmd/tidemark
cp README.md "dist/$name/"
tar -czf "dist/$name.tar.gz" -C dist "$name"
(cd dist && shasum -a 256 "$name.tar.gz" > "$name.tar.gz.sha256")

gh release create "$tag" --draft --target main --title "$tag" --notes "$(cat <<EOF
Apple Silicon build of \`tidemark\`. Needs macOS on Apple Silicon and mactop (\`brew install mactop\`).

\`\`\`sh
curl -fsSL https://github.com/lukeemerson/tidemark/releases/latest/download/$name.tar.gz | tar -xz
mv $name/tidemark ~/.local/bin/    # or anywhere on your PATH
\`\`\`

Downloaded in a browser instead? Clear the quarantine flag once: \`xattr -d com.apple.quarantine ~/.local/bin/tidemark\`.
EOF
)" "dist/$name.tar.gz" "dist/$name.tar.gz.sha256"

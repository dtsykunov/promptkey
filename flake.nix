{
  description = "PromptKey dev shell";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
    in {
      devShells.${system}.default = pkgs.mkShell {
        buildInputs = [
          pkgs.go
          pkgs.nodejs
          pkgs.wails
        ];

        shellHook = ''
          install_hook() {
            mkdir -p .git/hooks
            cat > .git/hooks/pre-commit << 'HOOK'
#!/usr/bin/env bash
set -e

STAGED=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' || true)
[ -z "$STAGED" ] && exit 0

UNFORMATTED=$(gofmt -l $STAGED)
if [ -n "$UNFORMATTED" ]; then
  echo "gofmt: unformatted files (run gofmt -w on each):"
  echo "$UNFORMATTED"
  exit 1
fi

# Find unique Go module roots for all staged .go files
MODULE_ROOTS=""
for f in $STAGED; do
  d=$(dirname "$f")
  while [ "$d" != "." ] && [ "$d" != "/" ]; do
    if [ -f "$d/go.mod" ]; then
      MODULE_ROOTS="$MODULE_ROOTS $d"
      break
    fi
    d=$(dirname "$d")
  done
  [ -f "go.mod" ] && MODULE_ROOTS="$MODULE_ROOTS ."
done
MODULE_ROOTS=$(echo "$MODULE_ROOTS" | tr ' ' '\n' | sort -u | grep -v '^$')

for dir in $MODULE_ROOTS; do
  echo "go vet: checking $dir"
  (cd "$dir" && go vet ./...) || exit 1
  echo "go build (windows/amd64): checking $dir"
  (cd "$dir" && GOOS=windows GOARCH=amd64 go build ./...) || exit 1
done
HOOK
            chmod +x .git/hooks/pre-commit
          }

          if [ -d .git ]; then
            install_hook
          fi
        '';
      };
    };
}

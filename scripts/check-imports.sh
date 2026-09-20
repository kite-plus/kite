#!/bin/sh
# Enforce the layering rule from docs/design/architecture.md: the domain core
# must not depend on any storage, rendering, runtime or publisher package.
#
# This check exists because the rule is load bearing and documentation alone
# does not stop an import from being added.
set -eu

MODULE=github.com/kite-plus/kite

# Packages that form the core, and the package prefixes they may never reach.
CORE="internal/content internal/schema internal/hook"
FORBIDDEN="internal/store internal/render internal/serve internal/api internal/publish internal/plugin internal/build internal/media internal/task"

status=0

for core in $CORE; do
    [ -d "$core" ] || continue

    deps=$(go list -deps "./$core/..." 2>/dev/null | grep "^$MODULE/" || true)

    for bad in $FORBIDDEN; do
        hit=$(printf '%s\n' "$deps" | grep "^$MODULE/$bad" || true)
        if [ -n "$hit" ]; then
            echo "FAIL: $core imports $bad"
            printf '%s\n' "$hit" | sed 's/^/        /'
            status=1
        fi
    done
done

if [ "$status" -eq 0 ]; then
    echo "import boundaries OK"
fi
exit "$status"

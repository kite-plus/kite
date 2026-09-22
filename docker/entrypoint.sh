#!/bin/sh
# Starts Kite in a container, creating the project first if the volume is
# empty.
#
# A container has nowhere to run `kite init`, so it is run here: an empty
# volume becomes a project with the defaults, and the rest of the
# installation -- what the site is called, where it lives, who owns it -- is
# asked for in the browser by the setup page. Nothing but that page answers
# until somebody has finished it.
set -eu

data=${KITE_DATA:-/data}
port=${KITE_PORT:-1717}

cd "$data"

if [ "$#" -eq 0 ]; then
    set -- serve
fi

# A command that is about kite rather than about a project does not get one
# made for it: asking a container its version should not leave a site behind.
case "$1" in
    version | help | completion | openapi | init | -h | --help) ;;
    *)
        if [ ! -f kite.yaml ]; then
            # No deploy workflow: a container is the deployment, and a GitHub
            # Pages workflow in a volume is a file nobody will ever run.
            kite init --yes --workflow=false . >/dev/null
            echo "kite: created a new project in $data"
        fi
        ;;
esac

# "serve" is expanded into the flags a container needs; every other command
# is run as it was given, so that `docker compose run --rm kite build` and
# `kite auth set-password` still work.
if [ "$1" = "serve" ]; then
    shift
    exec kite serve --admin --write --addr "0.0.0.0:$port" "$@"
fi
exec kite "$@"

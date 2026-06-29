#!/bin/sh
set -e

# Bootstrap a fresh data volume on first run. The files/ directory is the only
# thing mounted as a persistent volume, so an absent files/dagobert.db means the
# volume is empty: there is no case data to corrupt, so it is safe to create and
# migrate the database automatically (`dagobert update` also verifies the
# bundled MITRE data, which is already baked into the image). A populated volume
# is left untouched — upgrades stay explicit via `docker compose run --rm app
# update`, guarded by the schema check in `dagobert server`.
if [ ! -f files/dagobert.db ]; then
    echo "First run: bootstrapping a fresh data volume (creating + migrating the database)..."
    dagobert update
fi

exec dagobert "$@"

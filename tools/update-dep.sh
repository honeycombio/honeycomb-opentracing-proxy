#!/bin/bash
set -euo pipefail

if ! [[ "$0" =~ "tools/update-dep.sh" ]]; then
    echo "Run tools/update-dep.sh from the repository root"
    exit 1
fi


if ! type glide &> /dev/null; then
    echo "Didn't find glide, installing via go get"
    go get github.com/Masterminds/glide
fi

if ! type glide-vc &> /dev/null; then
    echo "Didn't find glide-vc; installing via go get"
    go get github.com/sgotti/glide-vc
fi

newdep=${1:-}

if [[ $newdep ]]; then
    echo "Installing new dependency $newdep with glide get"
    glide get $newdep --strip-vendor
    # Glide pulls a full repository clone into vendor/. This is inconvenient,
    # because, for example, we only need a few subpackages of
    # cloud.google.com/go. If we pull in the whole thing, we also need to pull
    # in all of its transitive dependencies. So we use glide-vc to remove
    # unneeded subpackages and tests out of vendor/.
    glide-vc --no-tests --only-code --use-lock-file --keep "**/*.proto"
else
    echo "Updating current dependencies"
    glide update --strip-vendor
    glide-vc --no-tests --only-code --use-lock-file --keep "**/*.proto"
fi

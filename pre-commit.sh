#!/usr/bin/env bash

echo "pre-commit.sh: Modifying main.go"

WD="$(pwd)"
sed -i -E \
    -e "s/version([ ]*)= \"(.*)\"/version\1= \"$(git describe --tags --abbrev=0)-$(git rev-parse --short HEAD)\"/" \
    -e "s/commit([ ]*)= \"(.*)\"/commit\1= \"$(git rev-parse HEAD)\"/" \
    -e "s/builtBy([ ]*)= \"(.*)\"/builtBy\1= \"golang\"/" \
    -e "s/date([ ]*)= \"(.*)\"/date\1= \"$(TZ=UTC date --rfc-3339=seconds)\"/" \
    -e "s/website([ ]*)= \"(.*)\"/website\1= \"https:\/\/sinashabani.dev\"/" \
    "$WD/main.go"

git add "$WD/main.go"


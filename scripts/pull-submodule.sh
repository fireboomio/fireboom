#!/usr/bin/env bash

cur_dir=$(
  cd "$(dirname "$0")" || exit
  pwd
)

function batch_cmd() {
  path=$cur_dir/$1
  branch=$2
  echo "prepare git pull from branch [$branch] for module [$path]"
  cd "$path" &&
    git fetch &&
    git checkout "$branch" &&
    git pull
}

batch_cmd "../assets/front" "release"
batch_cmd "../wundergraphGitSubmodule" "main"
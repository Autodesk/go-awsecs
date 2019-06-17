#!/usr/bin/env bash

set -e
set -u
set -o pipefail
set -x

cd cmd

for dir in in *;
do
  if [[ -d "${dir}" ]]; then
    cd "${dir}"
    go get
    # shellcheck disable=SC2034
    linux_ext=""
    # shellcheck disable=SC2034
    darwin_ext=""
    # shellcheck disable=SC2034
    windows_ext=.exe
    for goos in linux windows darwin; do
      # shellcheck disable=SC2043
      for goarch in amd64; do
        ext="${goos}_ext"
        out_no_ext="${dir}-${goos}-${goarch}"
        out="${dir}${!ext:-}"
        zip="${out_no_ext}.zip"
        rm -f "${out}" "${zip}"
        GOOS="${goos}" GOARCH="${goarch}" go build -o "${out}"
        zip "${zip}" "${out}"
        mv -vf "${zip}" ../../
        rm -f "${out}" "${zip}"
      done
    done
    cd ..
  fi
done

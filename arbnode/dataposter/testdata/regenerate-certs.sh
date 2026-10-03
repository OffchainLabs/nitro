#!/bin/bash
set -eu
cd "$(dirname "$0")"
for name in localhost client; do
    openssl genrsa -out "$name.key" 2048
    csr="$(openssl req -new -key "$name.key" -config "$name.cnf" -batch)"
    openssl x509 -req -signkey "$name.key" -out "$name.crt" -days 36500 -extensions req_ext -extfile "$name.cnf" <<< "$csr"
done

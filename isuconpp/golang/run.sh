#!/bin/bash

cd $(dirname $0)/src/main
ISUCONP_DB_NAME=isuconp \
ISUCONP_DB_USER=isucon \
ISUCONP_DB_PASSWORD=isucon \
../../bin/main -bind ":8080"

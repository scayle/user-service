#!/usr/bin/env bash

# IMPORTANT:
# 1. The output binary has to be /app
# 2. You have to use -gcflags="all=-N -l" to create the debug symbols
go build -gcflags="all=-N -l" -o /app

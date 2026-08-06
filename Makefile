SHELL := /bin/bash

# 只驗某一個 Task：make test TAGS=task1
TAGS ?= task1,task2,task3,task4

.PHONY: gen test

gen:
	@mockery

test: gen
	@go test -cover -tags $(TAGS) ./...

PROJECT_NAME := makefmt
PROJECT_OWNER := donaldgifford
DESCRIPTION := GNU Make formatter
PROJECT_URL := https://github.com/foo/bar

GO ?= go
GO_PACKAGE := github.com/foo/bar
GOOS ?= $(shell go env GOOS)

# This comment breaks the group
SINGLE_VAR := alone

A := 1
AB := 2
ABC := 3

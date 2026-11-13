PROJECT_NAME := "oxidized-exporte"

PREK_SKIP ?=

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

all: setup run ## run everything

test: lint unit ## Run all tests

unit: ## run unit tests
	go test ./...

setup: ## install required modules
	python -m pip install -U -r requirements-dev.txt
	prek install

run: ## run project
	python -m $(PROJECT_NAME)

build-docker: ## build docker image
	docker build -t $(PROJECT_NAME):test .

lint: ## run linters via prek
	prek run --all-files $(PREK_SKIP)

prek: ## run prek on changed files
	prek run

PROJECT_NAME := "oxidized-exporte"

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

all: setup run ## run everything

test: pre-commit-all unit integration ## Run all tests

setup: ## install required modules
	python -m pip install -U -r requirements-dev.txt
	pre-commit install

run: ## run project
	python -m $(PROJECT_NAME)

build-docker: ## build docker image
	docker build -t $(PROJECT_NAME):test .

pre-commit-all: ## run pre-commit on all files
	pre-commit run --all-files

pre-commit: ## run pre-commit
	pre-commit run

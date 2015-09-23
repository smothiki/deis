include ../../includes.mk

# the filepath to this repository, relative to $GOPATH/src
repo_path = github.com/deis/deis/builder


build: check-docker
	docker build -t smothiki/slugbuilder .

test-style:
# display output, then check
	$(GOFMT) $(GO_PACKAGES) $(GO_FILES)
	@$(GOFMT) $(GO_PACKAGES) $(GO_FILES) | read; if [ $$? == 0 ]; then echo "gofmt check failed."; exit 1; fi
	$(GOVET) $(repo_path) $(GO_PACKAGES_REPO_PATH)
	$(GOLINT) ./...

OUT := deploy-flamenco-manager
PKG := gitlab.com/blender-institute/azure-go-test
VERSION := $(shell git describe --tags --dirty --always)
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
STATIC_OUT := ${OUT}-v${VERSION}
PACKAGE_PATH := dist/${OUT}-${VERSION}

ifndef PACKAGE_PATH
# ${PACKAGE_PATH} is used in 'rm' commands, so it's important to check.
$(error PACKAGE_PATH is not set)
endif

all: binary

binary:
	go build -i -v -o ${OUT} -ldflags="-X main.applicationVersion=${VERSION}" ${PKG}

install:
	go install -i -v -ldflags="-X main.applicationVersion=${VERSION}" ${PKG}

version:
	@echo "Package: ${PKG}"
	@echo "Version: ${VERSION}"

test:
	go test -short ${PKG_LIST}

vet:
	@go vet ${PKG_LIST}

lint:
	@for file in ${GO_FILES} ;  do \
		golint $$file ; \
	done

run: binary
	./${OUT}

clean:
	@go clean -i -x
	rm -f ${OUT}-v*

static: vet lint
	go build -i -v -o ${STATIC_OUT} -tags netgo -ldflags="-extldflags \"-static\" -w -s -X main.applicationVersion=${VERSION}" ${PKG}

package:
	@$(MAKE) _prepare_package
	@$(MAKE) _package_linux
	@$(MAKE) _package_windows
	@$(MAKE) _package_darwin
	@$(MAKE) _finish_package

package_linux:
	@$(MAKE) _prepare_package
	@$(MAKE) _package_linux
	@$(MAKE) _finish_package

package_windows:
	@$(MAKE) _prepare_package
	@$(MAKE) _package_windows
	@$(MAKE) _finish_package

package_darwin:
	@$(MAKE) _prepare_package
	@$(MAKE) _package_darwin
	@$(MAKE) _finish_package

_package_linux:
	@$(MAKE) --no-print-directory GOOS=linux MONGOOS=linux GOARCH=amd64 STATIC_OUT=${PACKAGE_PATH}/${OUT} _package_tar

_package_windows:
	@$(MAKE) --no-print-directory GOOS=windows MONGOOS=windows GOARCH=amd64 STATIC_OUT=${PACKAGE_PATH}/${OUT}.exe _package_zip

_package_darwin:
	@$(MAKE) --no-print-directory GOOS=darwin MONGOOS=osx GOARCH=amd64 STATIC_OUT=${PACKAGE_PATH}/${OUT} _package_zip

_prepare_package:
	rm -rf ${PACKAGE_PATH}
	mkdir -p ${PACKAGE_PATH}
	cp -ua README.md LICENSE demo ${PACKAGE_PATH}/

_finish_package:
	rm -r ${PACKAGE_PATH}
	rm -f ${PACKAGE_PATH}.sha256
	sha256sum ${PACKAGE_PATH}* | tee ${PACKAGE_PATH}.sha256

_package_tar: static
	tar -C $(dir ${PACKAGE_PATH}) -zcf $(PWD)/${PACKAGE_PATH}-${GOOS}.tar.gz $(notdir ${PACKAGE_PATH})
	rm ${STATIC_OUT}

_package_zip: static
	cd $(dir ${PACKAGE_PATH}) && zip -9 -r -q $(notdir ${PACKAGE_PATH})-${GOOS}.zip $(notdir ${PACKAGE_PATH})
	rm ${STATIC_OUT}

.PHONY: run binary version static vet lint deploy package
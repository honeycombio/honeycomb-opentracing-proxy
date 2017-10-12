
PKG := honeycombio/zipkinproxy
VERSION := $(shell git describe --tags --always --dirty)

DOTFILE_IMAGE = $(subst :,_,$(subst /,_,$(PKG))-$(VERSION))

all: container

# Build container
container: .container-$(DOTFILE_IMAGE) container-name
.container-$(DOTFILE_IMAGE):
	@docker build -t $(PKG):$(VERSION) .
	@docker images -q $(IMAGE):$(VERSION) > $@

container-name:
	@echo "container: $(PKG):$(VERSION)"

# Publish container
push: .container-$(DOTFILE_IMAGE)
	@docker push $(PKG):$(VERSION)

# Publish container tagged with 'head'
push-head: .container-$(DOTFILE_IMAGE)
	@docker tag $(PKG):$(VERSION) $(PKG):head
	@docker push $(PKG):head
	@echo "pushed: $(PKG):head"

clean:
	rm -r .container-*

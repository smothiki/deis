include ../includes.mk

TEMPLATE_IMAGES=master slave
BUILT_IMAGES=base zk marathon $(TEMPLATE_IMAGES)

ZK_IMAGE = $(IMAGE_PREFIX)mesos-zk:$(BUILD_TAG)
ZK_DEV_IMAGE = $(DEV_REGISTRY)/$(ZK_IMAGE)
MASTER_IMAGE = $(IMAGE_PREFIX)mesos-master:$(BUILD_TAG)
MASTER_DEV_IMAGE = $(DEV_REGISTRY)/$(MASTER_IMAGE)
MARATHON_IMAGE = $(IMAGE_PREFIX)mesos-marathon:$(BUILD_TAG)
MARATHON_DEV_IMAGE = $(DEV_REGISTRY)/$(MARATHON_IMAGE)
SLAVE_IMAGE = $(IMAGE_PREFIX)mesos-slave:$(BUILD_TAG)
SLAVE_DEV_IMAGE = $(DEV_REGISTRY)/$(SLAVE_IMAGE)

build: check-docker
	docker build -t deis/mesos-zk:$(BUILD_TAG) zk/
	docker build -t deis/mesos-base:$(BUILD_TAG) base/
	$(foreach I, $(TEMPLATE_IMAGES), \
		sed -e "s/#FROM is generated dynamically by the Makefile/FROM deis\/mesos-base:${BUILD_TAG}/" $(I)/Dockerfile.template > $(I)/Dockerfile ; \
		docker build -t deis/mesos-$(I):$(BUILD_TAG) $(I)/ && \
		rm $(I)/Dockerfile ; \
	)
	docker build -t deis/mesos-marathon:$(BUILD_TAG) marathon/

clean: check-docker check-registry
	$(foreach I, $(BUILT_IMAGES), \
		docker rmi deis/mesos-$(I):$(BUILD_TAG) ; \
		docker rmi $(REGISTRY)/deis/mesos-$(I):$(BUILD_TAG) ; \
	)

full-clean: check-docker check-registry
	$(foreach I, $(BUILT_IMAGES), \
		docker images -q deis/mesos-$(I) | xargs docker rmi -f ; \
		docker images -q $(REGISTRY)/deis/mesos-$(I) | xargs docker rmi -f ; \
	)

install: check-deisctl
	deisctl install mesos-zk
	deisctl install mesos-master
	deisctl install mesos-marathon
	deisctl install mesos-slave

uninstall: check-deisctl
	deisctl uninstall mesos-slave
	deisctl uninstall mesos-marathon
	deisctl uninstall mesos-master
	deisctl uninstall mesos-zk

start: check-deisctl
	deisctl start mesos-zk
	deisctl start mesos-master
	deisctl start mesos-marathon
	deisctl start mesos-slave

stop: check-deisctl
	deisctl stop mesos-slave
	deisctl stop mesos-master
	deisctl stop mesos-marathon
	deisctl stop mesos-zk

restart: stop start

run: install start

dev-release: push set-image

push: check-registry
	docker tag $(ZK_IMAGE) $(ZK_DEV_IMAGE)
	docker push $(ZK_DEV_IMAGE)
	docker tag $(MASTER_IMAGE) $(MASTER_DEV_IMAGE)
	docker push $(MASTER_DEV_IMAGE)
	docker tag $(MARATHON_IMAGE) $(MARATHON_DEV_IMAGE)
	docker push $(MARATHON_DEV_IMAGE)
	docker tag $(SLAVE_IMAGE) $(SLAVE_DEV_IMAGE)
	docker push $(SLAVE_DEV_IMAGE)

set-image: check-deisctl
	deisctl config mesos-zk set image=$(ZK_DEV_IMAGE)
	deisctl config mesos-master set image=$(MASTER_DEV_IMAGE)
	deisctl config mesos-marathon set image=$(MARATHON_DEV_IMAGE)
	deisctl config mesos-slave set image=$(SLAVE_DEV_IMAGE)

release:
	docker push $(ZK_IMAGE)
	docker push $(MASTER_IMAGE)
	docker push $(MARATHON_IMAGE)
	docker push $(SLAVE_IMAGE)

deploy: build dev-release restart

test: test-unit test-functional

test-unit:
	@echo no unit tests

test-functional:
	@echo no functional tests

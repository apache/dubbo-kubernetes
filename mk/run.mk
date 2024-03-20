NUM_OF_DATAPLANES ?= 100
NUM_OF_SERVICES ?= 80
DISTRIBUTION_TARGET_NAME ?= $(PROJECT_NAME)
DUBBO_CP_ADDRESS ?= grpcs://localhost:5678
DISTRIBUTION_FOLDER=build/distributions/$(GOOS)-$(GOARCH)/$(DISTRIBUTION_TARGET_NAME)

CP_STORE = memory
CP_ENV += DUBBO_ENVIRONMENT=universal DUBBO_MULTIZONE_ZONE_NAME=zone-1 DUBBO_STORE_TYPE=$(CP_STORE)

.PHONY: run/xds-client
run/xds-client:
	go run ./tools/xds-client/... run --dps "${NUM_OF_DATAPLANES}" --services "${NUM_OF_SERVICES}" --xds-server-address "${DUBBO_CP_ADDRESS}"

.PHONY: run/dubbo-cp
run/dubbo-cp:
	go run ./app/dubbo-cp/... run --log-level=debug

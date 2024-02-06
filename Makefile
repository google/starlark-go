BIN_DIR='bin/'

.PHONY: build
build:
	mkdir -p ${BIN_DIR} && go build -o ${BIN_DIR} ./...

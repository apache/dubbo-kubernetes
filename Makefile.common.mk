tidy-go:
	@find -name go.mod -execdir go mod tidy \;

mod-download-go:
	@-GOFLAGS="-mod=readonly" find -name go.mod -execdir go mod download \;
# go mod tidy is needed with Golang 1.16+ as go mod download affects go.sum
# https://github.com/golang/go/issues/43994
	@find -name go.mod -execdir go mod tidy \;

format-go: tidy-go
	@${FINDFILES} -name '*.go' \( ! \( -name '*.gen.go' -o -name '*.pb.go' \) \) -print0 | ${XARGS} common/scripts/format_go.sh

.PHONY: format-go tidy-go mod-download-go

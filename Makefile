PROTOC_GEN_GO_PKG=github.com/golang/protobuf/protoc-gen-go
PKGMAP=Mgoogle/protobuf/descriptor.proto=$(PROTOC_GEN_GO_PKG)/descriptor

gen:
	protoc -I. --go_out=$(PKGMAP):protos *.proto

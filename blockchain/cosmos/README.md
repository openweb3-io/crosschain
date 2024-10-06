
## Install plugins
```
# go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@v1.16.0
go install github.com/cosmos/gogoproto/protoc-gen-gocosmos
```

## Updating protobuf

Crosschain should work with all cosmos chains, but some of them are not fully backwards compatible
with the latest cosmos-sdk (which crosschain will be on).  To support the incompatible features,
we must manually generate different clients using protobuf defining the different interfaces of some chains.

To generate the protobuf, you need to have [`buf`](https://github.com/bufbuild/buf) installed. Then run:

```
./generate-proto.sh
```
// proto 模块：服务间共享的 gRPC API 契约（生成代码）
// 该模块仅依赖 grpc / protobuf 运行时，不包含任何业务逻辑。
module github.com/museflow/proto

go 1.26

toolchain go1.26.6

require (
	google.golang.org/grpc v1.72.3
	google.golang.org/protobuf v1.36.10
)

require (
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250218202821-56aae31c358a // indirect
)

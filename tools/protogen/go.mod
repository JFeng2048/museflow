// protogen 是独立的开发工具模块，刻意不加入 go.work，
// 以免把 protoreflect 等开发期依赖带入业务模块。
module github.com/museflow/protogen

go 1.26

require (
	github.com/jhump/protoreflect v1.18.1
	google.golang.org/protobuf v1.36.12
)

require (
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/jhump/protoreflect/v2 v2.0.0-beta.1 // indirect
	github.com/petermattis/goid v0.0.0-20260113132338-7c7de50cc741 // indirect
	golang.org/x/sync v0.20.0 // indirect
)

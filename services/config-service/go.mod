module github.com/museflow/config-service

go 1.26

toolchain go1.26.6

require (
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.10.9
	github.com/museflow/pkg/envloader v0.0.0
	github.com/museflow/pkg/logger v0.0.0
	github.com/museflow/proto v0.0.0
	google.golang.org/grpc v1.72.3
	gorm.io/driver/postgres v1.5.9
	gorm.io/gorm v1.30.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20231201235250-de7065d80cb9 // indirect
	github.com/jackc/pgx/v5 v5.5.5 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/crypto v0.33.0 // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/sync v0.11.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250218202821-56aae31c358a // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)

// 本地开发通过 go.work 解析；replace 保证单模块构建时也能找到 proto 契约
replace github.com/museflow/proto => ../../proto

replace github.com/museflow/pkg/envloader => ../../pkg/envloader

replace github.com/museflow/pkg/logger => ../../pkg/logger

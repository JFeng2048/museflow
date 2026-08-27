# MuseFlow 根目录构建脚本
#
# 仓库为 monorepo，各服务独立 go.mod，根目录无 go.mod；
# 本地开发通过 go.work 统一解析模块。

GATEWAY_DIR := services/api-gateway
USER_DIR    := services/user-service
PROTO_DIR   := proto

GATEWAY_PKG := github.com/museflow/api-gateway/...
USER_PKG    := github.com/museflow/user-service/...
PROTO_PKG   := github.com/museflow/proto/...

BIN_DIR := bin

.PHONY: help
help: ## 显示可用命令
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

.PHONY: init
init: ## 初始化 go.work 并安装开发工具
	go work sync
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/air-verse/air@latest

.PHONY: proto
proto: ## 生成 gRPC 代码
	bash scripts/gen-proto.sh

.PHONY: swagger
swagger: ## 生成 api-gateway 的 Swagger 文档
	cd $(GATEWAY_DIR) && swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal

.PHONY: tidy
tidy: ## 整理各模块依赖
	cd $(PROTO_DIR) && go mod tidy
	cd $(USER_DIR) && go mod tidy
	cd $(GATEWAY_DIR) && go mod tidy

.PHONY: build
build: build-gateway build-user ## 编译全部服务

.PHONY: build-gateway
build-gateway: ## 编译 api-gateway
	cd $(GATEWAY_DIR) && go build -o ../../$(BIN_DIR)/api-gateway ./cmd/server

.PHONY: build-user
build-user: ## 编译 user-service
	cd $(USER_DIR) && go build -o ../../$(BIN_DIR)/user-service ./cmd/server

.PHONY: run-gateway
run-gateway: ## 本地启动 api-gateway
	cd $(GATEWAY_DIR) && go run ./cmd/server

.PHONY: run-user
run-user: ## 本地启动 user-service
	cd $(USER_DIR) && go run ./cmd/server

.PHONY: watch
watch: ## 同时热重载两个服务（需先 make init 安装 air）
	@echo "启动 user-service(5002) 与 api-gateway(5001) 热重载..."
	@(cd $(USER_DIR) && air) & (cd $(GATEWAY_DIR) && air)

.PHONY: watch-gateway
watch-gateway: ## 仅热重载 api-gateway
	cd $(GATEWAY_DIR) && air

.PHONY: watch-user
watch-user: ## 仅热重载 user-service
	cd $(USER_DIR) && air

.PHONY: test
test: ## 运行全部测试
	go test $(USER_PKG) $(GATEWAY_PKG)

.PHONY: vet
vet: ## 静态检查
	go vet $(USER_PKG) $(GATEWAY_PKG) $(PROTO_PKG)

.PHONY: fmt
fmt: ## 格式化代码
	gofmt -w $(PROTO_DIR) $(USER_DIR) $(GATEWAY_DIR)

.PHONY: docker
docker: ## 构建两个服务的镜像（上下文为仓库根目录）
	docker build -f $(GATEWAY_DIR)/Dockerfile -t museflow/api-gateway:latest .
	docker build -f $(USER_DIR)/Dockerfile -t museflow/user-service:latest .

.PHONY: clean
clean: ## 清理构建产物
	rm -rf $(BIN_DIR)

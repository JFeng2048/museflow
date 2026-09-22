package client

import (
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	modelpb "github.com/museflow/proto/model"
)

// ModelClient 封装 config-service 的 gRPC 连接。
//
// 与 UserClient 采用同一套连接策略：grpc.ClientConn 内部自带连接池与自动重连，
// 全局复用单个实例即可，不需要自行实现池化。
type ModelClient struct {
	conn   *grpc.ClientConn
	client modelpb.ModelServiceClient
}

// NewModelClient 建立到 config-service 的连接。
//
// 使用非阻塞式 Dial：连接在首次 RPC 时惰性建立，
// 避免 config-service 尚未就绪时网关启动失败。
func NewModelClient(target string) (*ModelClient, error) {
	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
	)
	if err != nil {
		return nil, fmt.Errorf("创建 config-service 连接失败: %w", err)
	}

	return &ModelClient{conn: conn, client: modelpb.NewModelServiceClient(conn)}, nil
}

// Service 返回 gRPC 存根。
func (c *ModelClient) Service() modelpb.ModelServiceClient {
	return c.client
}

// Close 关闭底层连接。
func (c *ModelClient) Close() error {
	return c.conn.Close()
}

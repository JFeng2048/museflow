// Package client 管理到后端微服务的 gRPC 连接。
package client

import (
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	userpb "github.com/museflow/proto/user"
)

// UserClient 封装 user-service 的 gRPC 连接。
//
// grpc.ClientConn 内部自带连接池与自动重连，
// 全局复用单个实例即可，不需要自行实现池化。
type UserClient struct {
	conn   *grpc.ClientConn
	client userpb.UserServiceClient
}

// NewUserClient 建立到 user-service 的连接。
//
// 使用非阻塞式 Dial：连接在首次 RPC 时惰性建立，
// 避免 user-service 尚未就绪时网关启动失败。
func NewUserClient(target string) (*UserClient, error) {
	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
		// 默认负载均衡策略，配合 DNS 解析可支持多实例
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
	)
	if err != nil {
		return nil, fmt.Errorf("创建 user-service 连接失败: %w", err)
	}

	return &UserClient{conn: conn, client: userpb.NewUserServiceClient(conn)}, nil
}

// Service 返回 gRPC 存根。
func (c *UserClient) Service() userpb.UserServiceClient {
	return c.client
}

// Close 关闭底层连接。
func (c *UserClient) Close() error {
	return c.conn.Close()
}

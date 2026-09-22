// Package handler 实现 config-service 的 gRPC 协议层。
//
// 与 user-service 的分层约定一致：本包只负责 proto 消息与领域模型之间的转换，
// 不承载业务规则；错误统一经 mapError 映射为 gRPC status，业务文案（中文）
// 直接透出，不把裸错误抛过 gRPC 边界。
package handler

import (
	modelpb "github.com/museflow/proto/model"

	"github.com/museflow/config-service/internal/service"
)

// ModelHandler 实现 modelpb.ModelServiceServer 的全部 RPC。
type ModelHandler struct {
	modelpb.UnimplementedModelServiceServer

	svc *service.Service
}

// NewModelHandler 构造系统配置域 gRPC 处理器。
func NewModelHandler(svc *service.Service) *ModelHandler {
	return &ModelHandler{svc: svc}
}

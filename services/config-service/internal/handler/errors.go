package handler

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/museflow/config-service/internal/repository"
	"github.com/museflow/config-service/internal/service"
)

// ErrInvalidUUID 请求里的 user_uuid 不是合法 UUID。
var ErrInvalidUUID = errors.New("用户标识格式非法")

// mapError 把业务错误统一映射为 gRPC status。
//
// 与 user-service 的约定一致：业务错误文案（中文）直接透出，裸错误不外泄。
// 归属校验失败（改别人的渠道）与资源不存在返回同一个 NotFound，
// 不向调用方暴露他人资源是否存在。
func mapError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ErrInvalidUUID), errors.Is(err, service.ErrInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())

	// 唯一约束冲突统一归 AlreadyExists：仓储层已按约束名分流文案，
	// 上层不需要再区分是渠道编码、模型编码还是同组内的调用标识。
	case errors.Is(err, repository.ErrProviderCodeExists),
		errors.Is(err, repository.ErrModelCodeExists),
		errors.Is(err, repository.ErrModelDuplicated),
		errors.Is(err, repository.ErrUserProviderNameExists):
		return status.Error(codes.AlreadyExists, err.Error())

	case errors.Is(err, repository.ErrProviderHasModels),
		errors.Is(err, repository.ErrUserProviderHasModels):
		// FailedPrecondition：资源被其他数据引用导致删除失败，调用方应先清理
		// 引用，重试同一个请求没有意义。
		return status.Error(codes.FailedPrecondition, err.Error())

	case errors.Is(err, service.ErrCredentialUnreadable):
		// FailedPrecondition：密文当前不可用，重试同一个请求没有意义，
		// 调用方应先重新填写密钥——错误文案本身就是给用户的下一步提示。
		return status.Error(codes.FailedPrecondition, err.Error())

	case errors.Is(err, repository.ErrProviderNotFound),
		errors.Is(err, repository.ErrModelNotFound),
		errors.Is(err, repository.ErrUserProviderNotFound),
		errors.Is(err, repository.ErrUserModelNotFound),
		errors.Is(err, repository.ErrSettingNotFound):
		return status.Error(codes.NotFound, err.Error())

	default:
		// 加密失败、数据库连接失败等：对调用方而言就是服务不可用，
		// 返回 Internal 并保留原始文案，便于与服务端日志比对。
		return status.Error(codes.Internal, err.Error())
	}
}

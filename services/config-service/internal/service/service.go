// Package service 实现 config-service 的业务逻辑。
//
// 分层与 user-service 一致：handler -> service -> repository，单向依赖。
// 本领域按表拆文件（provider / model / user / setting / available）而不建子包：
// 「用户端可用模型」要合并平台模型与用户自定义模型，五张表共用一个 Service
// 才能顺理成章地做这种跨表视图；拆子包只会让仓储在包间来回传递。
package service

import (
	"github.com/museflow/config-service/internal/pkg/secret"
	"github.com/museflow/config-service/internal/repository"
)

// Service 系统配置域服务。
type Service struct {
	providers     repository.ProviderRepository
	models        repository.ModelRepository
	userProviders repository.UserProviderRepository
	userModels    repository.UserModelRepository
	settings      repository.SettingRepository

	// secrets 凭证加解密器，只用于写入侧加密。
	// 本服务没有任何解密下发路径，因此响应里永远只出现 api_key_hint。
	secrets *secret.Encrypter
}

// Deps 组装 Service 所需的依赖。
type Deps struct {
	Providers     repository.ProviderRepository
	Models        repository.ModelRepository
	UserProviders repository.UserProviderRepository
	UserModels    repository.UserModelRepository
	Settings      repository.SettingRepository
	Secrets       *secret.Encrypter
}

// NewService 组装系统配置域服务。
func NewService(deps Deps) *Service {
	return &Service{
		providers:     deps.Providers,
		models:        deps.Models,
		userProviders: deps.UserProviders,
		userModels:    deps.UserModels,
		settings:      deps.Settings,
		secrets:       deps.Secrets,
	}
}

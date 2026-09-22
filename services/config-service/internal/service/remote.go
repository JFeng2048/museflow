package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/museflow/config-service/internal/model"
	"github.com/museflow/config-service/internal/pkg/upstream"
)

// FetchRemoteInput 拉取上游模型目录的入参。
//
// 三种定位方式与 proto 的约定一致，按优先级取第一个非空的：
// ProviderID（平台渠道）-> UserProviderID + UserUUID（用户渠道）-> BaseURL + APIKey（临时探测）。
type FetchRemoteInput struct {
	ProviderID     int64
	UserProviderID int64
	UserUUID       uuid.UUID
	BaseURL        string
	APIKey         string
	Protocol       string
}

// RemoteModel 上游模型目录条目，供后台勾选后批量登记。
type RemoteModel struct {
	ID        string
	Object    string
	OwnedBy   string
	CreatedAt string
}

// FetchProviderModels 拉取某个渠道的模型目录。
//
// 这是本服务唯一会「出网」的路径：管理员在后台填完 base_url 与密钥后，
// 先当场探测能不能连通、有哪些模型，再决定要不要落库。
func (s *Service) FetchProviderModels(ctx context.Context, in FetchRemoteInput) ([]RemoteModel, error) {
	opt, err := s.resolveUpstream(ctx, in)
	if err != nil {
		return nil, err
	}

	items, err := s.upstream.ListModels(ctx, opt)
	if err != nil {
		return nil, err
	}

	out := make([]RemoteModel, 0, len(items))
	for _, item := range items {
		out = append(out, RemoteModel{
			ID:        item.ID,
			Object:    item.Object,
			OwnedBy:   item.OwnedBy,
			CreatedAt: item.CreatedAt,
		})
	}
	return out, nil
}

// resolveUpstream 定位本次要访问的上游，产出归一化后的请求参数。
func (s *Service) resolveUpstream(ctx context.Context, in FetchRemoteInput) (upstream.Options, error) {
	switch {
	case in.ProviderID > 0:
		return s.upstreamFromPlatformProvider(ctx, in.ProviderID)

	case in.UserProviderID > 0:
		// 归属标识缺失只可能是调用方没带登录态，此时放行会拿 uuid.Nil 去匹配，
		// 任何渠道都不会命中，最终报「渠道不存在」，与「未登录」难以区分。
		if err := validateUserUUID(in.UserUUID); err != nil {
			return upstream.Options{}, err
		}
		return s.upstreamFromUserProvider(ctx, in.UserProviderID, in.UserUUID)

	case strings.TrimSpace(in.BaseURL) != "":
		return s.upstreamFromInline(in)

	default:
		return upstream.Options{}, invalidArgumentf("provider_id、user_provider_id、base_url 必须提供一个")
	}
}

// upstreamFromPlatformProvider 用平台渠道的落库配置发起探测。
//
// 停用中的渠道同样允许探测：停用只是不再对用户端可见，后台维护时
// 仍需要能拉到目录来核对模型清单。
func (s *Service) upstreamFromPlatformProvider(ctx context.Context, id int64) (upstream.Options, error) {
	provider, err := s.providers.GetByID(ctx, id)
	if err != nil {
		return upstream.Options{}, err
	}

	key, err := s.openAPIKey(provider.APIKeyCiphertext)
	if err != nil {
		return upstream.Options{}, err
	}
	return upstream.Options{
		Protocol: provider.Protocol,
		BaseURL:  provider.BaseURL,
		APIKey:   key,
	}, nil
}

// upstreamFromUserProvider 用用户自定义渠道的落库配置发起探测。
//
// 走 GetOwnedByID 而不是 GetByID：归属校验放在数据层，上层漏传 user_uuid
// 也只会得到「渠道不存在」，碰不到别人的渠道。
func (s *Service) upstreamFromUserProvider(ctx context.Context, id int64, userUUID uuid.UUID) (upstream.Options, error) {
	provider, err := s.userProviders.GetOwnedByID(ctx, id, userUUID)
	if err != nil {
		return upstream.Options{}, err
	}

	key, err := s.openAPIKey(provider.APIKeyCiphertext)
	if err != nil {
		return upstream.Options{}, err
	}
	return upstream.Options{
		Protocol: provider.Protocol,
		BaseURL:  provider.BaseURL,
		APIKey:   key,
	}, nil
}

// upstreamFromInline 用表单里临时填写的 base_url 与密钥发起探测。
//
// 这条路径的存在理由：渠道还没有保存，.api_key_hint 也无从比对，
// 「先确认能连通再落库」比「保存失败再删」对用户更友好。
func (s *Service) upstreamFromInline(in FetchRemoteInput) (upstream.Options, error) {
	baseURL, err := validateBaseURL(in.BaseURL)
	if err != nil {
		return upstream.Options{}, err
	}
	protocol, err := normalizeInlineProtocol(in.Protocol)
	if err != nil {
		return upstream.Options{}, err
	}
	if err := validateAPIKey(in.APIKey); err != nil {
		return upstream.Options{}, err
	}

	return upstream.Options{
		Protocol: protocol,
		BaseURL:  baseURL,
		APIKey:   strings.TrimSpace(in.APIKey),
	}, nil
}

// normalizeInlineProtocol 归一化临时探测的协议，空值按 openai 处理。
//
// 只有临时探测允许协议缺省：落库时 protocol 是必填项，而手工填 base_url 的场景里，
// 操作者通常只知道自己填的是 OpenAI 兼容地址，逼他选协议只会增加一步无意义操作。
func normalizeInlineProtocol(protocol string) (string, error) {
	if strings.TrimSpace(protocol) == "" {
		return model.ProtocolOpenAI, nil
	}
	return validateProtocol(protocol)
}

// openAPIKey 解出渠道密钥明文，仅用于组装本次请求头。
//
// 明文不写日志、不进响应、不落任何缓存：解密结果的生命周期覆盖不到一次 HTTP 调用。
// 密文为空表示该渠道未配置密钥（本地 Ollama 一类），按匿名访问处理。
func (s *Service) openAPIKey(ciphertext string) (string, error) {
	if strings.TrimSpace(ciphertext) == "" {
		return "", nil
	}

	key, err := s.secrets.Decrypt(ciphertext)
	if err != nil {
		// 不包装底层错误：secret 包的两种失败（格式非法 / 解密失败）对调用方
		// 都是「密文已不可用」，包装只会让用户看到两遍同一件事。
		return "", ErrCredentialUnreadable
	}
	return key, nil
}

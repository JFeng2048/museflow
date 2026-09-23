package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/museflow/config-service/internal/model"
	"github.com/museflow/config-service/internal/repository"
)

// createModelFakes 覆盖 CreateModel 走到仓储之前需要的两个依赖。
// 其余依赖留空：这条路径不碰用户模型与系统配置。
type createModelFakes struct {
	provider *stubProviderRepo
	models   *stubModelRepo
}

func newCreateModelFakes() *createModelFakes {
	return &createModelFakes{
		provider: &stubProviderRepo{provider: &model.ModelProvider{ID: 1, Code: "agnes"}},
		models:   &stubModelRepo{},
	}
}

func (f *createModelFakes) service() *Service {
	return NewService(Deps{
		Providers: f.provider,
		Models:    f.models,
	})
}

// stubProviderRepo 只按测试需要返回渠道，其余方法留零值。
type stubProviderRepo struct {
	provider *model.ModelProvider
}

func (r *stubProviderRepo) List(context.Context, repository.ProviderListFilter) ([]model.ModelProvider, int64, error) {
	return nil, 0, nil
}

func (r *stubProviderRepo) GetByID(context.Context, int64) (*model.ModelProvider, error) {
	return r.provider, nil
}

func (r *stubProviderRepo) Create(context.Context, *model.ModelProvider) error { return nil }

func (r *stubProviderRepo) Update(context.Context, int64, map[string]any) error {
	return nil
}

func (r *stubProviderRepo) Delete(context.Context, int64) error { return nil }

func (r *stubProviderRepo) ExistsByCode(context.Context, string) (bool, error) {
	return false, nil
}

// stubModelRepo 记录 Create 的调用，用于断言「字段没通过校验就不该落库」。
type stubModelRepo struct {
	created  *model.Model
	createAt int
}

func (r *stubModelRepo) List(context.Context, repository.ModelListFilter) ([]model.Model, int64, error) {
	return nil, 0, nil
}

func (r *stubModelRepo) GetByID(context.Context, int64) (*model.Model, error) {
	return nil, repository.ErrModelNotFound
}

func (r *stubModelRepo) Create(_ context.Context, m *model.Model) error {
	r.createAt++
	r.created = m
	return nil
}

func (r *stubModelRepo) Update(context.Context, int64, map[string]any) error { return nil }

func (r *stubModelRepo) Delete(context.Context, int64) error { return nil }

func (r *stubModelRepo) ListPlatformActive(context.Context, string) ([]model.Model, error) {
	return nil, nil
}

func modelInput(code string) ModelInput {
	return ModelInput{
		ProviderID: 1,
		Code:       code,
		Name:       "agnes-3.0-flash",
		ModelType:  "chat",
		APIModel:   "agnes-3.0-flash",
	}
}

// TestCreateModelCodeWidth 回归 model.code 的列宽校验。
//
// 模型表的 code 是 varchar(50)，这里曾用 maxAPIModelLen（100）校验 code，
// 51~100 字符的编码能通过参数校验、到 Postgres 才报 22001，用户看到的是 500。
func TestCreateModelCodeWidth(t *testing.T) {
	exact := strings.Repeat("a", maxCodeLen)
	tooLong := strings.Repeat("a", maxCodeLen+1)

	t.Run("等于列宽上限时落库", func(t *testing.T) {
		fakes := newCreateModelFakes()
		m, err := fakes.service().CreateModel(context.Background(), modelInput(exact))
		if err != nil {
			t.Fatalf("CreateModel() 返回错误: %v", err)
		}
		if fakes.models.createAt != 1 || m.Code != exact {
			t.Fatalf("Create 调用次数 = %d，code = %q", fakes.models.createAt, m.Code)
		}
	})

	t.Run("超过列宽上限时拒绝且不落库", func(t *testing.T) {
		fakes := newCreateModelFakes()
		_, err := fakes.service().CreateModel(context.Background(), modelInput(tooLong))
		if !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("错误 = %v，期望 ErrInvalidArgument", err)
		}
		if !strings.Contains(err.Error(), "code") {
			t.Fatalf("错误信息 %q 未指明字段 code", err.Error())
		}
		if fakes.models.createAt != 0 {
			t.Fatalf("校验失败却调用了 Create：%d 次", fakes.models.createAt)
		}
	})
}

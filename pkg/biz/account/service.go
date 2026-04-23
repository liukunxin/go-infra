package account

import (
	"context"
	"errors"
	"time"
)

// Service 账号服务，封装常用账号管理操作。
//
// 使用前需注入 AccountStore、BindingStore 与 IDGenerator 的具体实现：
//
//	svc := account.NewService(myAccountStore, myBindingStore, myIDGen)
type Service struct {
	accounts AccountStore
	bindings BindingStore
	idGen    IDGenerator
}

// NewService 创建账号服务。
func NewService(accounts AccountStore, bindings BindingStore, idGen IDGenerator) *Service {
	return &Service{
		accounts: accounts,
		bindings: bindings,
		idGen:    idGen,
	}
}

// FindOrCreate 通过登录凭证查找账号，不存在则自动创建新账号并建立绑定关系。
//
// 返回值 created=true 表示本次调用新建了账号。
//
// 注意：本方法不是原子操作。在高并发注册场景下，同一凭证可能触发并发创建。
// 建议业务侧在 BindingStore.Bind 中依赖数据库唯一索引捕获冲突，并对 ErrBindingConflict 进行重试。
func (s *Service) FindOrCreate(ctx context.Context, loginType, credential string) (acc *Account, created bool, err error) {
	binding, err := s.bindings.FindByCredential(ctx, loginType, credential)
	if err != nil && !errors.Is(err, ErrBindingNotFound) {
		return nil, false, err
	}

	if err == nil {
		// 已有绑定，直接查账号
		acc, err = s.accounts.GetByID(ctx, binding.AccountID)
		if err != nil {
			return nil, false, err
		}
		if err = checkStatus(acc); err != nil {
			return nil, false, err
		}
		return acc, false, nil
	}

	// 未找到绑定，创建新账号
	now := time.Now()
	acc = &Account{
		ID:        s.idGen.Generate(),
		Status:    StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err = s.accounts.Create(ctx, acc); err != nil {
		return nil, false, err
	}
	b := &LoginBinding{
		AccountID:  acc.ID,
		LoginType:  loginType,
		Credential: credential,
		CreatedAt:  now,
	}
	if err = s.bindings.Bind(ctx, b); err != nil {
		return nil, false, err
	}
	return acc, true, nil
}

// GetByID 查询账号，自动校验状态。
func (s *Service) GetByID(ctx context.Context, id string) (*Account, error) {
	acc, err := s.accounts.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err = checkStatus(acc); err != nil {
		return nil, err
	}
	return acc, nil
}

// Bind 为已有账号绑定新的登录方式。
//
// 若该凭证已绑定其他账号，返回 ErrBindingConflict。
// 若该账号同类型已绑定，返回 ErrBindingExists。
func (s *Service) Bind(ctx context.Context, accountID, loginType, credential string) error {
	// 检查凭证是否已被其他账号绑定
	existing, err := s.bindings.FindByCredential(ctx, loginType, credential)
	if err != nil && !errors.Is(err, ErrBindingNotFound) {
		return err
	}
	if err == nil && existing.AccountID != accountID {
		return ErrBindingConflict
	}
	if err == nil && existing.AccountID == accountID {
		return ErrBindingExists
	}

	return s.bindings.Bind(ctx, &LoginBinding{
		AccountID:  accountID,
		LoginType:  loginType,
		Credential: credential,
		CreatedAt:  time.Now(),
	})
}

// Unbind 解绑指定账号的某种登录方式。
//
// 注意：业务侧应确保解绑后账号至少保留一种可用登录方式，避免账号无法登录。
func (s *Service) Unbind(ctx context.Context, accountID, loginType string) error {
	return s.bindings.Unbind(ctx, accountID, loginType)
}

// ListBindings 列出账号已绑定的所有登录方式。
func (s *Service) ListBindings(ctx context.Context, accountID string) ([]*LoginBinding, error) {
	return s.bindings.ListByAccountID(ctx, accountID)
}

// Freeze 冻结账号，禁止登录（数据保留）。
func (s *Service) Freeze(ctx context.Context, accountID string) error {
	return s.accounts.UpdateStatus(ctx, accountID, StatusFrozen)
}

// Unfreeze 解冻账号，恢复正常状态。
func (s *Service) Unfreeze(ctx context.Context, accountID string) error {
	return s.accounts.UpdateStatus(ctx, accountID, StatusActive)
}

// Delete 注销账号（软删除）。注销后账号不可登录，但数据仍保留于数据库。
func (s *Service) Delete(ctx context.Context, accountID string) error {
	return s.accounts.UpdateStatus(ctx, accountID, StatusDeleted)
}

// checkStatus 校验账号状态，冻结/注销时返回对应错误。
func checkStatus(acc *Account) error {
	switch acc.Status {
	case StatusFrozen:
		return ErrAccountFrozen
	case StatusDeleted:
		return ErrAccountDeleted
	}
	return nil
}

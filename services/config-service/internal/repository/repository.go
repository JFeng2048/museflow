// Package repository 提供数据访问层，屏蔽底层 GORM 细节。
//
// 与 user-service 的两点差异：
//  1. schema 由 database/config_svc.sql 维护，本层不执行 AutoMigrate；
//  2. 表名带 config_svc 前缀，由 model 包的 TableName() 指定。
//
// 跨域只做逻辑关联（user_uuid 指向 user_svc.user.uuid），不建物理外键，
// 因此删除前是否需要连带清理，都由调用方显式决定。
package repository

import (
	"errors"
	"strings"

	"github.com/lib/pq"
)

// trimmed 去掉首尾空白，用于统一各仓储对字符串入参的处理。
func trimmed(s string) string {
	return strings.TrimSpace(s)
}

// uniqueConstraintName 从唯一约束冲突错误中取出约束名。
//
// 一张表上可能有多个唯一索引（如 model 既约束 code 全局唯一，又约束
// (provider_id, api_model) 组内唯一），文案只有指名道姓才能告诉用户该改哪个字段。
// pgx 与 lib/pq 的错误文本里都包含 unique constraint "xxx"，因此按文本提取即可。
func uniqueConstraintName(err error) string {
	const marker = `unique constraint "`

	msg := err.Error()
	start := strings.Index(msg, marker)
	if start < 0 {
		return ""
	}
	rest := msg[start+len(marker):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// likePattern 把关键字包成 ILIKE 模式，并按需转义通配符。
//
// 用户输入 % / _ 时不转义会被当通配符，导致「匹配到不该匹配的记录」，
// 这类行为很难在测试里暴露，因此在入口就转掉。
func likePattern(keyword string) string {
	escaped := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(keyword)
	return "%" + escaped + "%"
}

// isUniqueViolation 判断错误是否为唯一约束冲突（SQLSTATE 23505）。
//
// gorm.io/driver/postgres 默认走 pgx 驱动，错误类型是 *pgconn.PgError；
// 配置为 lib/pq 时是 *pq.Error。这里先识别 pq.Error，再兜一层 SQLSTATE 文本
// 判断：驱动组合变化时漏判的唯一冲突会被当成 500 返回，比多一层字符串匹配更难排查。
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}

	msg := err.Error()
	return strings.Contains(msg, "SQLSTATE 23505") ||
		strings.Contains(msg, "duplicate key value")
}

// mapUniqueViolation 把唯一约束冲突翻译成领域错误，其他错误原样返回。
func mapUniqueViolation(err error, conflict error) error {
	if err == nil {
		return nil
	}
	if isUniqueViolation(err) {
		return conflict
	}
	return err
}

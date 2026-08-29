-- 迁移 0001：为 user_svc.user 补充 2FA（双因素认证）相关字段
-- 对应 GORM 模型 User 的 MFAEnabled / MFASecret / MFARecoveryCodes
-- 执行前请确认 user_svc.user 表已存在且缺少以下列。

ALTER TABLE "user_svc"."user"
  ADD COLUMN IF NOT EXISTS "mfa_enabled" bool NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS "mfa_secret" varchar(100),
  ADD COLUMN IF NOT EXISTS "mfa_recovery_codes" text[];

COMMENT ON COLUMN "user_svc"."user"."mfa_enabled" IS '是否已启用双因素认证(TOTP)';
COMMENT ON COLUMN "user_svc"."user"."mfa_secret" IS 'TOTP 密钥(明文)，仅服务端可见，启用前为空';
COMMENT ON COLUMN "user_svc"."user"."mfa_recovery_codes" IS '恢复码数组，单次使用，始终保持 8 个';

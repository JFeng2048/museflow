/*
 Navicat Premium Dump SQL

 Source Server         : localhost
 Source Server Type    : PostgreSQL
 Source Server Version : 180006 (180006)
 Source Host           : localhost:5432
 Source Catalog        : museflow
 Source Schema         : user_svc

 Target Server Type    : PostgreSQL
 Target Server Version : 180006 (180006)
 File Encoding         : 65001

 Date: 27/08/2026 18:31:13
*/


-- ----------------------------
-- Sequence structure for audit_logs_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "user_svc"."audit_logs_id_seq";
CREATE SEQUENCE "user_svc"."audit_logs_id_seq" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;

-- ----------------------------
-- Sequence structure for user_oauths_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "user_svc"."user_oauths_id_seq";
CREATE SEQUENCE "user_svc"."user_oauths_id_seq" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;

-- ----------------------------
-- Sequence structure for user_sessions_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "user_svc"."user_sessions_id_seq";
CREATE SEQUENCE "user_svc"."user_sessions_id_seq" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;

-- ----------------------------
-- Sequence structure for users_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "user_svc"."users_id_seq";
CREATE SEQUENCE "user_svc"."users_id_seq" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;

-- ----------------------------
-- Table structure for audit_logs
-- ----------------------------
DROP TABLE IF EXISTS "user_svc"."audit_logs";
CREATE TABLE "user_svc"."audit_logs" (
  "id" int8 NOT NULL DEFAULT nextval('"user_svc".audit_logs_id_seq'::regclass),
  "user_uuid" uuid,
  "action" varchar(100) COLLATE "pg_catalog"."default" NOT NULL,
  "resource" varchar(50) COLLATE "pg_catalog"."default" NOT NULL,
  "resource_id" varchar(100) COLLATE "pg_catalog"."default",
  "ip" inet,
  "user_agent" text COLLATE "pg_catalog"."default",
  "detail" jsonb,
  "created_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
COMMENT ON COLUMN "user_svc"."audit_logs"."id" IS '自增主键';
COMMENT ON COLUMN "user_svc"."audit_logs"."user_uuid" IS '操作人UUID，NULL表示系统操作';
COMMENT ON COLUMN "user_svc"."audit_logs"."action" IS '操作类型: login/logout/create_user/update_user/delete_user/assign_role等';
COMMENT ON COLUMN "user_svc"."audit_logs"."resource" IS '操作资源类型';
COMMENT ON COLUMN "user_svc"."audit_logs"."resource_id" IS '操作资源ID';
COMMENT ON COLUMN "user_svc"."audit_logs"."ip" IS '请求IP';
COMMENT ON COLUMN "user_svc"."audit_logs"."user_agent" IS '客户端信息';
COMMENT ON COLUMN "user_svc"."audit_logs"."detail" IS '详细数据，JSONB格式';
COMMENT ON COLUMN "user_svc"."audit_logs"."created_at" IS '操作时间';
COMMENT ON TABLE "user_svc"."audit_logs" IS '用户操作审计日志表';

-- ----------------------------
-- Table structure for permissions
-- ----------------------------
DROP TABLE IF EXISTS "user_svc"."permissions";
CREATE TABLE "user_svc"."permissions" (
  "id" int2 NOT NULL,
  "code" varchar(100) COLLATE "pg_catalog"."default" NOT NULL,
  "name" varchar(100) COLLATE "pg_catalog"."default" NOT NULL,
  "resource" varchar(50) COLLATE "pg_catalog"."default" NOT NULL,
  "action" varchar(50) COLLATE "pg_catalog"."default" NOT NULL,
  "description" text COLLATE "pg_catalog"."default",
  "created_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
COMMENT ON COLUMN "user_svc"."permissions"."id" IS '权限ID';
COMMENT ON COLUMN "user_svc"."permissions"."code" IS '权限编码: user:read, novel:publish等';
COMMENT ON COLUMN "user_svc"."permissions"."name" IS '权限名称';
COMMENT ON COLUMN "user_svc"."permissions"."resource" IS '资源类型: user/novel/chapter/material/publish/system/hotspot';
COMMENT ON COLUMN "user_svc"."permissions"."action" IS '操作类型: read/write/delete/publish/admin';
COMMENT ON COLUMN "user_svc"."permissions"."description" IS '权限描述';
COMMENT ON TABLE "user_svc"."permissions" IS '权限定义表';

-- ----------------------------
-- Table structure for role_permissions
-- ----------------------------
DROP TABLE IF EXISTS "user_svc"."role_permissions";
CREATE TABLE "user_svc"."role_permissions" (
  "role_id" int2 NOT NULL,
  "permission_id" int2 NOT NULL,
  "created_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
COMMENT ON COLUMN "user_svc"."role_permissions"."role_id" IS '关联roles.id';
COMMENT ON COLUMN "user_svc"."role_permissions"."permission_id" IS '关联permissions.id';
COMMENT ON COLUMN "user_svc"."role_permissions"."created_at" IS '授权时间';
COMMENT ON TABLE "user_svc"."role_permissions" IS '角色-权限关联表';

-- ----------------------------
-- Table structure for roles
-- ----------------------------
DROP TABLE IF EXISTS "user_svc"."roles";
CREATE TABLE "user_svc"."roles" (
  "id" int2 NOT NULL,
  "code" varchar(50) COLLATE "pg_catalog"."default" NOT NULL,
  "name" varchar(100) COLLATE "pg_catalog"."default" NOT NULL,
  "description" text COLLATE "pg_catalog"."default",
  "is_system" bool NOT NULL DEFAULT false,
  "created_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
COMMENT ON COLUMN "user_svc"."roles"."id" IS '角色ID';
COMMENT ON COLUMN "user_svc"."roles"."code" IS '角色编码: super_admin/admin/editor/viewer/user';
COMMENT ON COLUMN "user_svc"."roles"."name" IS '角色名称';
COMMENT ON COLUMN "user_svc"."roles"."description" IS '角色描述';
COMMENT ON COLUMN "user_svc"."roles"."is_system" IS '是否系统内置角色';
COMMENT ON TABLE "user_svc"."roles" IS '角色定义表';

-- ----------------------------
-- Table structure for user_oauths
-- ----------------------------
DROP TABLE IF EXISTS "user_svc"."user_oauths";
CREATE TABLE "user_svc"."user_oauths" (
  "id" int8 NOT NULL DEFAULT nextval('"user_svc".user_oauths_id_seq'::regclass),
  "user_uuid" uuid NOT NULL,
  "provider" varchar(50) COLLATE "pg_catalog"."default" NOT NULL,
  "provider_user_id" varchar(255) COLLATE "pg_catalog"."default" NOT NULL,
  "provider_email" varchar(255) COLLATE "pg_catalog"."default",
  "provider_nickname" varchar(100) COLLATE "pg_catalog"."default",
  "provider_avatar" varchar(500) COLLATE "pg_catalog"."default",
  "access_token" text COLLATE "pg_catalog"."default",
  "refresh_token" text COLLATE "pg_catalog"."default",
  "expires_at" timestamp(6),
  "extra" jsonb,
  "is_active" bool NOT NULL DEFAULT true,
  "last_login_at" timestamp(6),
  "created_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
COMMENT ON COLUMN "user_svc"."user_oauths"."id" IS '自增主键';
COMMENT ON COLUMN "user_svc"."user_oauths"."user_uuid" IS '关联user_svc.users.uuid';
COMMENT ON COLUMN "user_svc"."user_oauths"."provider" IS '第三方平台: github/google/wechat/qq/apple等';
COMMENT ON COLUMN "user_svc"."user_oauths"."provider_user_id" IS '第三方平台用户唯一标识';
COMMENT ON COLUMN "user_svc"."user_oauths"."provider_email" IS '第三方邮箱(快照)';
COMMENT ON COLUMN "user_svc"."user_oauths"."provider_nickname" IS '第三方昵称(快照)';
COMMENT ON COLUMN "user_svc"."user_oauths"."provider_avatar" IS '第三方头像URL(快照)';
COMMENT ON COLUMN "user_svc"."user_oauths"."access_token" IS 'Access Token，加密存储';
COMMENT ON COLUMN "user_svc"."user_oauths"."refresh_token" IS 'Refresh Token，加密存储';
COMMENT ON COLUMN "user_svc"."user_oauths"."expires_at" IS 'Token过期时间';
COMMENT ON COLUMN "user_svc"."user_oauths"."extra" IS '平台特有字段: openid/unionid/scope等，JSONB格式';
COMMENT ON COLUMN "user_svc"."user_oauths"."is_active" IS '是否有效';
COMMENT ON COLUMN "user_svc"."user_oauths"."last_login_at" IS '最后一次通过此平台登录的时间';
COMMENT ON COLUMN "user_svc"."user_oauths"."created_at" IS '绑定时间';
COMMENT ON COLUMN "user_svc"."user_oauths"."updated_at" IS '更新时间';
COMMENT ON TABLE "user_svc"."user_oauths" IS '第三方登录关联表';

-- ----------------------------
-- Table structure for user_roles
-- ----------------------------
DROP TABLE IF EXISTS "user_svc"."user_roles";
CREATE TABLE "user_svc"."user_roles" (
  "user_uuid" uuid NOT NULL,
  "role_id" int2 NOT NULL,
  "granted_by" uuid,
  "granted_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
COMMENT ON COLUMN "user_svc"."user_roles"."user_uuid" IS '关联user_svc.users.uuid';
COMMENT ON COLUMN "user_svc"."user_roles"."role_id" IS '关联roles.id';
COMMENT ON COLUMN "user_svc"."user_roles"."granted_by" IS '授权人UUID';
COMMENT ON COLUMN "user_svc"."user_roles"."granted_at" IS '授权时间';
COMMENT ON TABLE "user_svc"."user_roles" IS '用户-角色关联表';

-- ----------------------------
-- Table structure for user_sessions
-- ----------------------------
DROP TABLE IF EXISTS "user_svc"."user_sessions";
CREATE TABLE "user_svc"."user_sessions" (
  "id" int8 NOT NULL DEFAULT nextval('"user_svc".user_sessions_id_seq'::regclass),
  "user_uuid" uuid NOT NULL,
  "session_token" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "login_ip" inet,
  "user_agent" text COLLATE "pg_catalog"."default",
  "expires_at" timestamp(6) NOT NULL,
  "is_revoked" bool NOT NULL DEFAULT false,
  "created_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
COMMENT ON COLUMN "user_svc"."user_sessions"."id" IS '自增主键';
COMMENT ON COLUMN "user_svc"."user_sessions"."user_uuid" IS '关联user_svc.users.uuid';
COMMENT ON COLUMN "user_svc"."user_sessions"."session_token" IS '会话令牌，存储在Redis和Cookie中';
COMMENT ON COLUMN "user_svc"."user_sessions"."login_ip" IS '登录IP';
COMMENT ON COLUMN "user_svc"."user_sessions"."user_agent" IS '设备信息';
COMMENT ON COLUMN "user_svc"."user_sessions"."expires_at" IS '会话过期时间';
COMMENT ON COLUMN "user_svc"."user_sessions"."is_revoked" IS '是否已撤销';
COMMENT ON COLUMN "user_svc"."user_sessions"."created_at" IS '登录时间';
COMMENT ON TABLE "user_svc"."user_sessions" IS '登录会话审计表';

-- ----------------------------
-- Table structure for users
-- ----------------------------
DROP TABLE IF EXISTS "user_svc"."users";
CREATE TABLE "user_svc"."users" (
  "id" int8 NOT NULL DEFAULT nextval('"user_svc".users_id_seq'::regclass),
  "uuid" uuid NOT NULL DEFAULT gen_random_uuid(),
  "email" varchar(255) COLLATE "pg_catalog"."default" NOT NULL,
  "phone" varchar(20) COLLATE "pg_catalog"."default",
  "password_hash" varchar(255) COLLATE "pg_catalog"."default",
  "nickname" varchar(100) COLLATE "pg_catalog"."default" NOT NULL,
  "avatar_url" varchar(500) COLLATE "pg_catalog"."default",
  "bio" text COLLATE "pg_catalog"."default",
  "mfa_enabled" bool NOT NULL DEFAULT false,
  "mfa_secret" varchar(100) COLLATE "pg_catalog"."default",
  "mfa_recovery_codes" text[] COLLATE "pg_catalog"."default",
  "status" int2 NOT NULL DEFAULT 1,
  "email_verified" bool NOT NULL DEFAULT false,
  "phone_verified" bool NOT NULL DEFAULT false,
  "last_login_at" timestamp(6),
  "last_login_ip" inet,
  "last_login_platform" varchar(50) COLLATE "pg_catalog"."default",
  "login_fail_count" int4 NOT NULL DEFAULT 0,
  "locked_until" timestamp(6),
  "created_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
COMMENT ON COLUMN "user_svc"."users"."id" IS '自增主键，内部使用';
COMMENT ON COLUMN "user_svc"."users"."uuid" IS '用户全局唯一标识，对外使用';
COMMENT ON COLUMN "user_svc"."users"."email" IS '用户邮箱，登录账号，全平台唯一';
COMMENT ON COLUMN "user_svc"."users"."phone" IS '用户手机号，全平台唯一';
COMMENT ON COLUMN "user_svc"."users"."password_hash" IS 'bcrypt加密密码，第三方登录用户可为空';
COMMENT ON COLUMN "user_svc"."users"."nickname" IS '用户昵称';
COMMENT ON COLUMN "user_svc"."users"."avatar_url" IS '用户头像URL';
COMMENT ON COLUMN "user_svc"."users"."bio" IS '个人简介';
COMMENT ON COLUMN "user_svc"."users"."mfa_enabled" IS '是否开启TOTP多因素认证';
COMMENT ON COLUMN "user_svc"."users"."mfa_secret" IS 'TOTP密钥，加密存储';
COMMENT ON COLUMN "user_svc"."users"."mfa_recovery_codes" IS 'MFA恢复码数组';
COMMENT ON COLUMN "user_svc"."users"."status" IS '状态: 1=正常, 2=冻结, 3=已注销, 4=待审核';
COMMENT ON COLUMN "user_svc"."users"."email_verified" IS '邮箱是否已验证';
COMMENT ON COLUMN "user_svc"."users"."phone_verified" IS '手机号是否已验证';
COMMENT ON COLUMN "user_svc"."users"."last_login_at" IS '最后登录时间';
COMMENT ON COLUMN "user_svc"."users"."last_login_ip" IS '最后登录IP';
COMMENT ON COLUMN "user_svc"."users"."last_login_platform" IS '最后登录平台: web/ios/android';
COMMENT ON COLUMN "user_svc"."users"."login_fail_count" IS '连续登录失败次数';
COMMENT ON COLUMN "user_svc"."users"."locked_until" IS '账号锁定截止时间';
COMMENT ON COLUMN "user_svc"."users"."created_at" IS '注册时间';
COMMENT ON COLUMN "user_svc"."users"."updated_at" IS '最后更新时间';
COMMENT ON TABLE "user_svc"."users" IS '用户主表';

-- ----------------------------
-- Function structure for trigger_set_updated_at
-- ----------------------------
DROP FUNCTION IF EXISTS "user_svc"."trigger_set_updated_at"();
CREATE FUNCTION "user_svc"."trigger_set_updated_at"()
  RETURNS "pg_catalog"."trigger" AS $BODY$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$BODY$
  LANGUAGE plpgsql VOLATILE
  COST 100;
COMMENT ON FUNCTION "user_svc"."trigger_set_updated_at"() IS '自动更新updated_at字段的触发器函数';

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
ALTER SEQUENCE "user_svc"."audit_logs_id_seq"
OWNED BY "user_svc"."audit_logs"."id";
SELECT setval('"user_svc"."audit_logs_id_seq"', 1, false);

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
ALTER SEQUENCE "user_svc"."user_oauths_id_seq"
OWNED BY "user_svc"."user_oauths"."id";
SELECT setval('"user_svc"."user_oauths_id_seq"', 1, false);

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
ALTER SEQUENCE "user_svc"."user_sessions_id_seq"
OWNED BY "user_svc"."user_sessions"."id";
SELECT setval('"user_svc"."user_sessions_id_seq"', 1, false);

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
ALTER SEQUENCE "user_svc"."users_id_seq"
OWNED BY "user_svc"."users"."id";
SELECT setval('"user_svc"."users_id_seq"', 1, false);

-- ----------------------------
-- Primary Key structure for table audit_logs
-- ----------------------------
ALTER TABLE "user_svc"."audit_logs" ADD CONSTRAINT "audit_logs_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Triggers structure for table permissions
-- ----------------------------
CREATE TRIGGER "trg_permissions_updated_at" BEFORE UPDATE ON "user_svc"."permissions"
FOR EACH ROW
EXECUTE PROCEDURE "user_svc"."trigger_set_updated_at"();

-- ----------------------------
-- Primary Key structure for table permissions
-- ----------------------------
ALTER TABLE "user_svc"."permissions" ADD CONSTRAINT "permissions_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Primary Key structure for table role_permissions
-- ----------------------------
ALTER TABLE "user_svc"."role_permissions" ADD CONSTRAINT "role_permissions_pkey" PRIMARY KEY ("role_id", "permission_id");

-- ----------------------------
-- Triggers structure for table roles
-- ----------------------------
CREATE TRIGGER "trg_roles_updated_at" BEFORE UPDATE ON "user_svc"."roles"
FOR EACH ROW
EXECUTE PROCEDURE "user_svc"."trigger_set_updated_at"();

-- ----------------------------
-- Primary Key structure for table roles
-- ----------------------------
ALTER TABLE "user_svc"."roles" ADD CONSTRAINT "roles_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Triggers structure for table user_oauths
-- ----------------------------
CREATE TRIGGER "trg_oauths_updated_at" BEFORE UPDATE ON "user_svc"."user_oauths"
FOR EACH ROW
EXECUTE PROCEDURE "user_svc"."trigger_set_updated_at"();

-- ----------------------------
-- Primary Key structure for table user_oauths
-- ----------------------------
ALTER TABLE "user_svc"."user_oauths" ADD CONSTRAINT "user_oauths_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Primary Key structure for table user_roles
-- ----------------------------
ALTER TABLE "user_svc"."user_roles" ADD CONSTRAINT "user_roles_pkey" PRIMARY KEY ("user_uuid", "role_id");

-- ----------------------------
-- Primary Key structure for table user_sessions
-- ----------------------------
ALTER TABLE "user_svc"."user_sessions" ADD CONSTRAINT "user_sessions_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Triggers structure for table users
-- ----------------------------
CREATE TRIGGER "trg_users_updated_at" BEFORE UPDATE ON "user_svc"."users"
FOR EACH ROW
EXECUTE PROCEDURE "user_svc"."trigger_set_updated_at"();

-- ----------------------------
-- Primary Key structure for table users
-- ----------------------------
ALTER TABLE "user_svc"."users" ADD CONSTRAINT "users_pkey" PRIMARY KEY ("id");

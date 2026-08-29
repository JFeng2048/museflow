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

 Date: 29/08/2026 14:07:25
*/


-- ----------------------------
-- Sequence structure for audit_log_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "user_svc"."audit_log_id_seq";
CREATE SEQUENCE "user_svc"."audit_log_id_seq" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;

-- ----------------------------
-- Sequence structure for audit_log_id_seq1
-- ----------------------------
DROP SEQUENCE IF EXISTS "user_svc"."audit_log_id_seq1";
CREATE SEQUENCE "user_svc"."audit_log_id_seq1" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;

-- ----------------------------
-- Sequence structure for oauth_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "user_svc"."oauth_id_seq";
CREATE SEQUENCE "user_svc"."oauth_id_seq" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;

-- ----------------------------
-- Sequence structure for oauth_id_seq1
-- ----------------------------
DROP SEQUENCE IF EXISTS "user_svc"."oauth_id_seq1";
CREATE SEQUENCE "user_svc"."oauth_id_seq1" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;

-- ----------------------------
-- Sequence structure for session_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "user_svc"."session_id_seq";
CREATE SEQUENCE "user_svc"."session_id_seq" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;

-- ----------------------------
-- Sequence structure for session_id_seq1
-- ----------------------------
DROP SEQUENCE IF EXISTS "user_svc"."session_id_seq1";
CREATE SEQUENCE "user_svc"."session_id_seq1" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;

-- ----------------------------
-- Sequence structure for user_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "user_svc"."user_id_seq";
CREATE SEQUENCE "user_svc"."user_id_seq" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;

-- ----------------------------
-- Sequence structure for user_id_seq1
-- ----------------------------
DROP SEQUENCE IF EXISTS "user_svc"."user_id_seq1";
CREATE SEQUENCE "user_svc"."user_id_seq1" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;

-- ----------------------------
-- Table structure for audit_log
-- ----------------------------
DROP TABLE IF EXISTS "user_svc"."audit_log";
CREATE TABLE "user_svc"."audit_log" (
  "id" int8 NOT NULL DEFAULT nextval('"user_svc".audit_log_id_seq1'::regclass),
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
COMMENT ON COLUMN "user_svc"."audit_log"."id" IS '自增主键';
COMMENT ON COLUMN "user_svc"."audit_log"."user_uuid" IS '操作人UUID，NULL表示系统操作';
COMMENT ON COLUMN "user_svc"."audit_log"."action" IS '操作类型: login/logout/create_user/update_user/delete_user/assign_role等';
COMMENT ON COLUMN "user_svc"."audit_log"."resource" IS '操作资源类型';
COMMENT ON COLUMN "user_svc"."audit_log"."resource_id" IS '操作资源ID';
COMMENT ON COLUMN "user_svc"."audit_log"."ip" IS '请求IP';
COMMENT ON COLUMN "user_svc"."audit_log"."user_agent" IS '客户端信息';
COMMENT ON COLUMN "user_svc"."audit_log"."detail" IS '详细数据，JSONB格式';
COMMENT ON COLUMN "user_svc"."audit_log"."created_at" IS '操作时间';
COMMENT ON TABLE "user_svc"."audit_log" IS '用户操作审计日志表';

-- ----------------------------
-- Records of audit_log
-- ----------------------------

-- ----------------------------
-- Table structure for oauth
-- ----------------------------
DROP TABLE IF EXISTS "user_svc"."oauth";
CREATE TABLE "user_svc"."oauth" (
  "id" int8 NOT NULL DEFAULT nextval('"user_svc".oauth_id_seq1'::regclass),
  "user_uuid" uuid NOT NULL,
  "provider" varchar(50) COLLATE "pg_catalog"."default" NOT NULL,
  "sso_provider" varchar(50) COLLATE "pg_catalog"."default",
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
COMMENT ON COLUMN "user_svc"."oauth"."id" IS '自增主键';
COMMENT ON COLUMN "user_svc"."oauth"."user_uuid" IS '关联user_svc.user.uuid';
COMMENT ON COLUMN "user_svc"."oauth"."provider" IS '第三方平台: github/google/wechat/qq/apple等';
COMMENT ON COLUMN "user_svc"."oauth"."sso_provider" IS '企业SSO提供商: ldap/oidc/saml等';
COMMENT ON COLUMN "user_svc"."oauth"."provider_user_id" IS '第三方平台用户唯一标识';
COMMENT ON COLUMN "user_svc"."oauth"."provider_email" IS '第三方邮箱(快照)';
COMMENT ON COLUMN "user_svc"."oauth"."provider_nickname" IS '第三方昵称(快照)';
COMMENT ON COLUMN "user_svc"."oauth"."provider_avatar" IS '第三方头像URL(快照)';
COMMENT ON COLUMN "user_svc"."oauth"."access_token" IS 'Access Token，加密存储';
COMMENT ON COLUMN "user_svc"."oauth"."refresh_token" IS 'Refresh Token，加密存储';
COMMENT ON COLUMN "user_svc"."oauth"."expires_at" IS 'Token过期时间';
COMMENT ON COLUMN "user_svc"."oauth"."extra" IS '平台特有字段: openid/unionid/scope等，JSONB格式';
COMMENT ON COLUMN "user_svc"."oauth"."is_active" IS '是否有效';
COMMENT ON COLUMN "user_svc"."oauth"."last_login_at" IS '最后一次通过此平台登录的时间';
COMMENT ON COLUMN "user_svc"."oauth"."created_at" IS '绑定时间';
COMMENT ON COLUMN "user_svc"."oauth"."updated_at" IS '更新时间';
COMMENT ON TABLE "user_svc"."oauth" IS '第三方登录关联表';

-- ----------------------------
-- Records of oauth
-- ----------------------------

-- ----------------------------
-- Table structure for permission
-- ----------------------------
DROP TABLE IF EXISTS "user_svc"."permission";
CREATE TABLE "user_svc"."permission" (
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
COMMENT ON COLUMN "user_svc"."permission"."id" IS '权限ID';
COMMENT ON COLUMN "user_svc"."permission"."code" IS '权限编码: user:read, novel:publish等';
COMMENT ON COLUMN "user_svc"."permission"."name" IS '权限名称';
COMMENT ON COLUMN "user_svc"."permission"."resource" IS '资源类型: user/novel/chapter/material/publish/system/hotspot';
COMMENT ON COLUMN "user_svc"."permission"."action" IS '操作类型: read/write/delete/publish/admin';
COMMENT ON COLUMN "user_svc"."permission"."description" IS '权限描述';
COMMENT ON TABLE "user_svc"."permission" IS '权限定义表';

-- ----------------------------
-- Records of permission
-- ----------------------------
INSERT INTO "user_svc"."permission" VALUES (1, 'user:read', '查看用户', 'user', 'read', NULL, '2026-08-29 14:06:00.785944', '2026-08-29 14:06:00.785944');
INSERT INTO "user_svc"."permission" VALUES (2, 'user:write', '编辑用户', 'user', 'write', NULL, '2026-08-29 14:06:00.785944', '2026-08-29 14:06:00.785944');
INSERT INTO "user_svc"."permission" VALUES (3, 'user:delete', '删除用户', 'user', 'delete', NULL, '2026-08-29 14:06:00.785944', '2026-08-29 14:06:00.785944');
INSERT INTO "user_svc"."permission" VALUES (4, 'user:admin', '管理用户', 'user', 'admin', NULL, '2026-08-29 14:06:00.785944', '2026-08-29 14:06:00.785944');
INSERT INTO "user_svc"."permission" VALUES (5, 'novel:read', '查看作品', 'novel', 'read', NULL, '2026-08-29 14:06:00.785944', '2026-08-29 14:06:00.785944');
INSERT INTO "user_svc"."permission" VALUES (6, 'novel:write', '创作作品', 'novel', 'write', NULL, '2026-08-29 14:06:00.785944', '2026-08-29 14:06:00.785944');
INSERT INTO "user_svc"."permission" VALUES (7, 'novel:delete', '删除作品', 'novel', 'delete', NULL, '2026-08-29 14:06:00.785944', '2026-08-29 14:06:00.785944');
INSERT INTO "user_svc"."permission" VALUES (8, 'novel:publish', '发布作品', 'novel', 'publish', NULL, '2026-08-29 14:06:00.785944', '2026-08-29 14:06:00.785944');
INSERT INTO "user_svc"."permission" VALUES (9, 'novel:admin', '管理所有作品', 'novel', 'admin', NULL, '2026-08-29 14:06:00.785944', '2026-08-29 14:06:00.785944');
INSERT INTO "user_svc"."permission" VALUES (10, 'material:read', '查看素材', 'material', 'read', NULL, '2026-08-29 14:06:00.785944', '2026-08-29 14:06:00.785944');
INSERT INTO "user_svc"."permission" VALUES (11, 'material:write', '管理素材', 'material', 'write', NULL, '2026-08-29 14:06:00.785944', '2026-08-29 14:06:00.785944');
INSERT INTO "user_svc"."permission" VALUES (12, 'publish:read', '查看发布', 'publish', 'read', NULL, '2026-08-29 14:06:00.785944', '2026-08-29 14:06:00.785944');
INSERT INTO "user_svc"."permission" VALUES (13, 'publish:write', '执行发布', 'publish', 'write', NULL, '2026-08-29 14:06:00.785944', '2026-08-29 14:06:00.785944');
INSERT INTO "user_svc"."permission" VALUES (14, 'publish:admin', '管理发布', 'publish', 'admin', NULL, '2026-08-29 14:06:00.785944', '2026-08-29 14:06:00.785944');
INSERT INTO "user_svc"."permission" VALUES (15, 'system:admin', '系统管理', 'system', 'admin', NULL, '2026-08-29 14:06:00.785944', '2026-08-29 14:06:00.785944');
INSERT INTO "user_svc"."permission" VALUES (16, 'hotspot:read', '查看热点', 'hotspot', 'read', NULL, '2026-08-29 14:06:00.785944', '2026-08-29 14:06:00.785944');
INSERT INTO "user_svc"."permission" VALUES (17, 'hotspot:write', '管理热点', 'hotspot', 'write', NULL, '2026-08-29 14:06:00.785944', '2026-08-29 14:06:00.785944');

-- ----------------------------
-- Table structure for role
-- ----------------------------
DROP TABLE IF EXISTS "user_svc"."role";
CREATE TABLE "user_svc"."role" (
  "id" int2 NOT NULL,
  "code" varchar(50) COLLATE "pg_catalog"."default" NOT NULL,
  "name" varchar(100) COLLATE "pg_catalog"."default" NOT NULL,
  "description" text COLLATE "pg_catalog"."default",
  "is_system" bool NOT NULL DEFAULT false,
  "created_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
COMMENT ON COLUMN "user_svc"."role"."id" IS '角色ID';
COMMENT ON COLUMN "user_svc"."role"."code" IS '角色编码: super_admin/admin/user';
COMMENT ON COLUMN "user_svc"."role"."name" IS '角色名称';
COMMENT ON COLUMN "user_svc"."role"."description" IS '角色描述';
COMMENT ON COLUMN "user_svc"."role"."is_system" IS '是否系统内置角色';
COMMENT ON TABLE "user_svc"."role" IS '角色定义表';

-- ----------------------------
-- Records of role
-- ----------------------------
INSERT INTO "user_svc"."role" VALUES (1, 'super_admin', '超级管理员', '拥有全部系统权限', 't', '2026-08-29 14:06:00.78449', '2026-08-29 14:06:00.78449');
INSERT INTO "user_svc"."role" VALUES (2, 'admin', '管理员', '管理用户和内容', 't', '2026-08-29 14:06:00.78449', '2026-08-29 14:06:00.78449');
INSERT INTO "user_svc"."role" VALUES (3, 'user', '普通用户', '可创作和发布', 't', '2026-08-29 14:06:00.78449', '2026-08-29 14:06:00.78449');

-- ----------------------------
-- Table structure for role_permission
-- ----------------------------
DROP TABLE IF EXISTS "user_svc"."role_permission";
CREATE TABLE "user_svc"."role_permission" (
  "role_id" int2 NOT NULL,
  "permission_id" int2 NOT NULL,
  "created_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
COMMENT ON COLUMN "user_svc"."role_permission"."role_id" IS '关联user_svc.role.id';
COMMENT ON COLUMN "user_svc"."role_permission"."permission_id" IS '关联user_svc.permission.id';
COMMENT ON COLUMN "user_svc"."role_permission"."created_at" IS '授权时间';
COMMENT ON TABLE "user_svc"."role_permission" IS '角色-权限关联表';

-- ----------------------------
-- Records of role_permission
-- ----------------------------
INSERT INTO "user_svc"."role_permission" VALUES (1, 1, '2026-08-29 14:06:00.787515');
INSERT INTO "user_svc"."role_permission" VALUES (1, 2, '2026-08-29 14:06:00.787515');
INSERT INTO "user_svc"."role_permission" VALUES (1, 3, '2026-08-29 14:06:00.787515');
INSERT INTO "user_svc"."role_permission" VALUES (1, 4, '2026-08-29 14:06:00.787515');
INSERT INTO "user_svc"."role_permission" VALUES (1, 5, '2026-08-29 14:06:00.787515');
INSERT INTO "user_svc"."role_permission" VALUES (1, 6, '2026-08-29 14:06:00.787515');
INSERT INTO "user_svc"."role_permission" VALUES (1, 7, '2026-08-29 14:06:00.787515');
INSERT INTO "user_svc"."role_permission" VALUES (1, 8, '2026-08-29 14:06:00.787515');
INSERT INTO "user_svc"."role_permission" VALUES (1, 9, '2026-08-29 14:06:00.787515');
INSERT INTO "user_svc"."role_permission" VALUES (1, 10, '2026-08-29 14:06:00.787515');
INSERT INTO "user_svc"."role_permission" VALUES (1, 11, '2026-08-29 14:06:00.787515');
INSERT INTO "user_svc"."role_permission" VALUES (1, 12, '2026-08-29 14:06:00.787515');
INSERT INTO "user_svc"."role_permission" VALUES (1, 13, '2026-08-29 14:06:00.787515');
INSERT INTO "user_svc"."role_permission" VALUES (1, 14, '2026-08-29 14:06:00.787515');
INSERT INTO "user_svc"."role_permission" VALUES (1, 15, '2026-08-29 14:06:00.787515');
INSERT INTO "user_svc"."role_permission" VALUES (1, 16, '2026-08-29 14:06:00.787515');
INSERT INTO "user_svc"."role_permission" VALUES (1, 17, '2026-08-29 14:06:00.787515');
INSERT INTO "user_svc"."role_permission" VALUES (3, 5, '2026-08-29 14:06:00.788852');
INSERT INTO "user_svc"."role_permission" VALUES (3, 6, '2026-08-29 14:06:00.788852');
INSERT INTO "user_svc"."role_permission" VALUES (3, 8, '2026-08-29 14:06:00.788852');
INSERT INTO "user_svc"."role_permission" VALUES (3, 10, '2026-08-29 14:06:00.788852');
INSERT INTO "user_svc"."role_permission" VALUES (3, 11, '2026-08-29 14:06:00.788852');
INSERT INTO "user_svc"."role_permission" VALUES (3, 12, '2026-08-29 14:06:00.788852');
INSERT INTO "user_svc"."role_permission" VALUES (3, 13, '2026-08-29 14:06:00.788852');
INSERT INTO "user_svc"."role_permission" VALUES (3, 16, '2026-08-29 14:06:00.788852');

-- ----------------------------
-- Table structure for session
-- ----------------------------
DROP TABLE IF EXISTS "user_svc"."session";
CREATE TABLE "user_svc"."session" (
  "id" int8 NOT NULL DEFAULT nextval('"user_svc".session_id_seq1'::regclass),
  "user_uuid" uuid NOT NULL,
  "token" varchar(64) COLLATE "pg_catalog"."default" NOT NULL,
  "login_ip" inet,
  "user_agent" text COLLATE "pg_catalog"."default",
  "expires_at" timestamp(6) NOT NULL,
  "is_revoked" bool NOT NULL DEFAULT false,
  "created_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
COMMENT ON COLUMN "user_svc"."session"."id" IS '自增主键';
COMMENT ON COLUMN "user_svc"."session"."user_uuid" IS '关联user_svc.user.uuid';
COMMENT ON COLUMN "user_svc"."session"."token" IS '会话令牌，存储在Redis和Cookie中';
COMMENT ON COLUMN "user_svc"."session"."login_ip" IS '登录IP';
COMMENT ON COLUMN "user_svc"."session"."user_agent" IS '设备信息';
COMMENT ON COLUMN "user_svc"."session"."expires_at" IS '会话过期时间';
COMMENT ON COLUMN "user_svc"."session"."is_revoked" IS '是否已撤销';
COMMENT ON COLUMN "user_svc"."session"."created_at" IS '登录时间';
COMMENT ON TABLE "user_svc"."session" IS '登录会话审计表';

-- ----------------------------
-- Records of session
-- ----------------------------

-- ----------------------------
-- Table structure for user
-- ----------------------------
DROP TABLE IF EXISTS "user_svc"."user";
CREATE TABLE "user_svc"."user" (
  "id" int8 NOT NULL DEFAULT nextval('"user_svc".user_id_seq1'::regclass),
  "uuid" uuid NOT NULL DEFAULT gen_random_uuid(),
  "email" varchar(255) COLLATE "pg_catalog"."default" NOT NULL,
  "phone" varchar(20) COLLATE "pg_catalog"."default",
  "password_hash" varchar(255) COLLATE "pg_catalog"."default",
  "nickname" varchar(100) COLLATE "pg_catalog"."default" NOT NULL,
  "avatar_url" varchar(500) COLLATE "pg_catalog"."default",
  "bio" text COLLATE "pg_catalog"."default",
  "status" int2 NOT NULL DEFAULT 1,
  "email_verified" bool NOT NULL DEFAULT false,
  "phone_verified" bool NOT NULL DEFAULT false,
  "last_login_at" timestamp(6),
  "last_login_ip" inet,
  "last_login_platform" varchar(50) COLLATE "pg_catalog"."default",
  "login_fail_count" int4 NOT NULL DEFAULT 0,
  "locked_until" timestamp(6),
  "mfa_enabled" bool NOT NULL DEFAULT false,
  "mfa_secret" varchar(100) COLLATE "pg_catalog"."default",
  "mfa_recovery_codes" text[],
  "created_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
COMMENT ON COLUMN "user_svc"."user"."id" IS '自增主键，内部使用';
COMMENT ON COLUMN "user_svc"."user"."uuid" IS '用户全局唯一标识，对外使用';
COMMENT ON COLUMN "user_svc"."user"."email" IS '用户邮箱，登录账号，全平台唯一';
COMMENT ON COLUMN "user_svc"."user"."phone" IS '用户手机号，全平台唯一';
COMMENT ON COLUMN "user_svc"."user"."password_hash" IS 'bcrypt加密密码，第三方登录用户可为空';
COMMENT ON COLUMN "user_svc"."user"."nickname" IS '用户昵称';
COMMENT ON COLUMN "user_svc"."user"."avatar_url" IS '用户头像URL';
COMMENT ON COLUMN "user_svc"."user"."bio" IS '个人简介';
COMMENT ON COLUMN "user_svc"."user"."status" IS '状态: 1=正常, 2=冻结, 3=已注销, 4=待审核';
COMMENT ON COLUMN "user_svc"."user"."email_verified" IS '邮箱是否已验证';
COMMENT ON COLUMN "user_svc"."user"."phone_verified" IS '手机号是否已验证';
COMMENT ON COLUMN "user_svc"."user"."last_login_at" IS '最后登录时间';
COMMENT ON COLUMN "user_svc"."user"."last_login_ip" IS '最后登录IP';
COMMENT ON COLUMN "user_svc"."user"."last_login_platform" IS '最后登录平台: web/ios/android';
COMMENT ON COLUMN "user_svc"."user"."login_fail_count" IS '连续登录失败次数';
COMMENT ON COLUMN "user_svc"."user"."locked_until" IS '账号锁定截止时间';
COMMENT ON COLUMN "user_svc"."user"."mfa_enabled" IS '是否已启用双因素认证(TOTP)';
COMMENT ON COLUMN "user_svc"."user"."mfa_secret" IS 'TOTP 密钥(明文)，仅服务端可见，启用前为空';
COMMENT ON COLUMN "user_svc"."user"."mfa_recovery_codes" IS '恢复码数组，单次使用，始终保持 8 个';
COMMENT ON COLUMN "user_svc"."user"."created_at" IS '注册时间';
COMMENT ON COLUMN "user_svc"."user"."updated_at" IS '最后更新时间';
COMMENT ON TABLE "user_svc"."user" IS '用户主表';

-- ----------------------------
-- Records of user
-- ----------------------------
INSERT INTO "user_svc"."user" VALUES (1, 'd1a2b3c4-1234-5678-90ab-cdef12345678', 'admin@museflow.com', NULL, '$2a$10$N9qo8uLOickgx2ZMRZoMy.Mr/.wZ6E2FvFqNtT1XKqVqKqVqKqVqK', '系统管理员', NULL, NULL, 1, 't', 'f', NULL, NULL, NULL, 0, NULL, '2026-08-29 14:06:00.790015', '2026-08-29 14:06:00.790015');
INSERT INTO "user_svc"."user" VALUES (2, 'e2b3c4d5-2345-6789-01bc-def234567890', 'user@museflow.com', NULL, '$2a$10$N9qo8uLOickgx2ZMRZoMy.Mr/.wZ6E2FvFqNtT1XKqVqKqVqKqVqK', '测试用户', NULL, NULL, 1, 't', 'f', NULL, NULL, NULL, 0, NULL, '2026-08-29 14:06:00.791474', '2026-08-29 14:06:00.791474');

-- ----------------------------
-- Table structure for user_role
-- ----------------------------
DROP TABLE IF EXISTS "user_svc"."user_role";
CREATE TABLE "user_svc"."user_role" (
  "user_uuid" uuid NOT NULL,
  "role_id" int2 NOT NULL,
  "granted_by" uuid,
  "granted_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
COMMENT ON COLUMN "user_svc"."user_role"."user_uuid" IS '关联user_svc.user.uuid';
COMMENT ON COLUMN "user_svc"."user_role"."role_id" IS '关联user_svc.role.id';
COMMENT ON COLUMN "user_svc"."user_role"."granted_by" IS '授权人UUID';
COMMENT ON COLUMN "user_svc"."user_role"."granted_at" IS '授权时间';
COMMENT ON TABLE "user_svc"."user_role" IS '用户-角色关联表';

-- ----------------------------
-- Records of user_role
-- ----------------------------
INSERT INTO "user_svc"."user_role" VALUES ('d1a2b3c4-1234-5678-90ab-cdef12345678', 1, NULL, '2026-08-29 14:06:00.792642');
INSERT INTO "user_svc"."user_role" VALUES ('e2b3c4d5-2345-6789-01bc-def234567890', 3, NULL, '2026-08-29 14:06:00.794102');

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
SELECT setval('"user_svc"."audit_log_id_seq"', 1, false);

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
ALTER SEQUENCE "user_svc"."audit_log_id_seq1"
OWNED BY "user_svc"."audit_log"."id";
SELECT setval('"user_svc"."audit_log_id_seq1"', 1, false);

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
SELECT setval('"user_svc"."oauth_id_seq"', 1, false);

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
ALTER SEQUENCE "user_svc"."oauth_id_seq1"
OWNED BY "user_svc"."oauth"."id";
SELECT setval('"user_svc"."oauth_id_seq1"', 1, false);

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
SELECT setval('"user_svc"."session_id_seq"', 1, false);

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
ALTER SEQUENCE "user_svc"."session_id_seq1"
OWNED BY "user_svc"."session"."id";
SELECT setval('"user_svc"."session_id_seq1"', 1, false);

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
SELECT setval('"user_svc"."user_id_seq"', 1, false);

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
ALTER SEQUENCE "user_svc"."user_id_seq1"
OWNED BY "user_svc"."user"."id";
SELECT setval('"user_svc"."user_id_seq1"', 2, true);

-- ----------------------------
-- Indexes structure for table audit_log
-- ----------------------------
CREATE INDEX "idx_audit_action" ON "user_svc"."audit_log" USING btree (
  "action" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE INDEX "idx_audit_created" ON "user_svc"."audit_log" USING btree (
  "created_at" "pg_catalog"."timestamp_ops" ASC NULLS LAST
);
CREATE INDEX "idx_audit_resource" ON "user_svc"."audit_log" USING btree (
  "resource" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE INDEX "idx_audit_user" ON "user_svc"."audit_log" USING btree (
  "user_uuid" "pg_catalog"."uuid_ops" ASC NULLS LAST
);

-- ----------------------------
-- Primary Key structure for table audit_log
-- ----------------------------
ALTER TABLE "user_svc"."audit_log" ADD CONSTRAINT "audit_log_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table oauth
-- ----------------------------
CREATE INDEX "idx_oauth_active" ON "user_svc"."oauth" USING btree (
  "is_active" "pg_catalog"."bool_ops" ASC NULLS LAST
);
CREATE INDEX "idx_oauth_provider" ON "user_svc"."oauth" USING btree (
  "provider" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE INDEX "idx_oauth_provider_user" ON "user_svc"."oauth" USING btree (
  "provider" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "provider_user_id" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE INDEX "idx_oauth_user" ON "user_svc"."oauth" USING btree (
  "user_uuid" "pg_catalog"."uuid_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table oauth
-- ----------------------------
CREATE TRIGGER "trg_oauth_updated_at" BEFORE UPDATE ON "user_svc"."oauth"
FOR EACH ROW
EXECUTE PROCEDURE "user_svc"."trigger_set_updated_at"();

-- ----------------------------
-- Primary Key structure for table oauth
-- ----------------------------
ALTER TABLE "user_svc"."oauth" ADD CONSTRAINT "oauth_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table permission
-- ----------------------------
CREATE UNIQUE INDEX "idx_permission_code" ON "user_svc"."permission" USING btree (
  "code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table permission
-- ----------------------------
CREATE TRIGGER "trg_permission_updated_at" BEFORE UPDATE ON "user_svc"."permission"
FOR EACH ROW
EXECUTE PROCEDURE "user_svc"."trigger_set_updated_at"();

-- ----------------------------
-- Primary Key structure for table permission
-- ----------------------------
ALTER TABLE "user_svc"."permission" ADD CONSTRAINT "permission_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table role
-- ----------------------------
CREATE UNIQUE INDEX "idx_role_code" ON "user_svc"."role" USING btree (
  "code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table role
-- ----------------------------
CREATE TRIGGER "trg_role_updated_at" BEFORE UPDATE ON "user_svc"."role"
FOR EACH ROW
EXECUTE PROCEDURE "user_svc"."trigger_set_updated_at"();

-- ----------------------------
-- Primary Key structure for table role
-- ----------------------------
ALTER TABLE "user_svc"."role" ADD CONSTRAINT "role_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Primary Key structure for table role_permission
-- ----------------------------
ALTER TABLE "user_svc"."role_permission" ADD CONSTRAINT "role_permission_pkey" PRIMARY KEY ("role_id", "permission_id");

-- ----------------------------
-- Indexes structure for table session
-- ----------------------------
CREATE INDEX "idx_session_expires" ON "user_svc"."session" USING btree (
  "expires_at" "pg_catalog"."timestamp_ops" ASC NULLS LAST
);
CREATE INDEX "idx_session_revoked" ON "user_svc"."session" USING btree (
  "is_revoked" "pg_catalog"."bool_ops" ASC NULLS LAST
);
CREATE INDEX "idx_session_token" ON "user_svc"."session" USING btree (
  "token" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE INDEX "idx_session_user" ON "user_svc"."session" USING btree (
  "user_uuid" "pg_catalog"."uuid_ops" ASC NULLS LAST
);

-- ----------------------------
-- Primary Key structure for table session
-- ----------------------------
ALTER TABLE "user_svc"."session" ADD CONSTRAINT "session_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table user
-- ----------------------------
CREATE UNIQUE INDEX "idx_user_email" ON "user_svc"."user" USING btree (
  "email" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "idx_user_phone" ON "user_svc"."user" USING btree (
  "phone" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE INDEX "idx_user_status" ON "user_svc"."user" USING btree (
  "status" "pg_catalog"."int2_ops" ASC NULLS LAST
);
CREATE INDEX "idx_user_uuid" ON "user_svc"."user" USING btree (
  "uuid" "pg_catalog"."uuid_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table user
-- ----------------------------
CREATE TRIGGER "trg_user_updated_at" BEFORE UPDATE ON "user_svc"."user"
FOR EACH ROW
EXECUTE PROCEDURE "user_svc"."trigger_set_updated_at"();

-- ----------------------------
-- Primary Key structure for table user
-- ----------------------------
ALTER TABLE "user_svc"."user" ADD CONSTRAINT "user_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table user_role
-- ----------------------------
CREATE INDEX "idx_user_role_role" ON "user_svc"."user_role" USING btree (
  "role_id" "pg_catalog"."int2_ops" ASC NULLS LAST
);
CREATE INDEX "idx_user_role_user" ON "user_svc"."user_role" USING btree (
  "user_uuid" "pg_catalog"."uuid_ops" ASC NULLS LAST
);

-- ----------------------------
-- Primary Key structure for table user_role
-- ----------------------------
ALTER TABLE "user_svc"."user_role" ADD CONSTRAINT "user_role_pkey" PRIMARY KEY ("user_uuid", "role_id");

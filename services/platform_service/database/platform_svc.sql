/*
 Navicat Premium Dump SQL

 Source Server         : localhost
 Source Server Type    : PostgreSQL
 Source Server Version : 180006 (180006)
 Source Host           : localhost:5432
 Source Catalog        : museflow
 Source Schema         : platform_svc

 Target Server Type    : PostgreSQL
 Target Server Version : 180006 (180006)
 File Encoding         : 65001

 Date: 02/09/2026 14:43:34
*/


-- ----------------------------
-- Sequence structure for fanqie_bindings_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "platform_svc"."fanqie_bindings_id_seq";
CREATE SEQUENCE "platform_svc"."fanqie_bindings_id_seq" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;

-- ----------------------------
-- Sequence structure for publish_records_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "platform_svc"."publish_records_id_seq";
CREATE SEQUENCE "platform_svc"."publish_records_id_seq" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;

-- ----------------------------
-- Sequence structure for qidian_bindings_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "platform_svc"."qidian_bindings_id_seq";
CREATE SEQUENCE "platform_svc"."qidian_bindings_id_seq" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;

-- ----------------------------
-- Sequence structure for qimao_bindings_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "platform_svc"."qimao_bindings_id_seq";
CREATE SEQUENCE "platform_svc"."qimao_bindings_id_seq" 
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;

-- ----------------------------
-- Table structure for fanqie_bindings
-- ----------------------------
DROP TABLE IF EXISTS "platform_svc"."fanqie_bindings";
CREATE TABLE "platform_svc"."fanqie_bindings" (
  "id" int8 NOT NULL DEFAULT nextval('"platform_svc".fanqie_bindings_id_seq'::regclass),
  "user_uuid" uuid NOT NULL,
  "platform_user_id" varchar(255) COLLATE "pg_catalog"."default",
  "cookie" text COLLATE "pg_catalog"."default" NOT NULL,
  "csrf_token" varchar(255) COLLATE "pg_catalog"."default",
  "expires_at" timestamp(6),
  "is_active" bool NOT NULL DEFAULT true,
  "last_sync_at" timestamp(6),
  "sync_status" varchar(50) COLLATE "pg_catalog"."default" DEFAULT 'pending'::character varying,
  "sync_error" text COLLATE "pg_catalog"."default",
  "created_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
COMMENT ON COLUMN "platform_svc"."fanqie_bindings"."id" IS '自增主键';
COMMENT ON COLUMN "platform_svc"."fanqie_bindings"."user_uuid" IS '关联 user_svc.users.uuid';
COMMENT ON COLUMN "platform_svc"."fanqie_bindings"."platform_user_id" IS '番茄作者ID，从后台获取';
COMMENT ON COLUMN "platform_svc"."fanqie_bindings"."cookie" IS '番茄作家后台的 Cookie，**需 AES-256-GCM 加密存储**';
COMMENT ON COLUMN "platform_svc"."fanqie_bindings"."csrf_token" IS 'X-Secsdk-Csrf-Token，从请求头获取，**需加密存储**';
COMMENT ON COLUMN "platform_svc"."fanqie_bindings"."expires_at" IS 'Cookie 预估过期时间（约 1-2 个月），到期需引导用户重新绑定';
COMMENT ON COLUMN "platform_svc"."fanqie_bindings"."is_active" IS '凭证是否可用: true=有效, false=已过期/失效';
COMMENT ON COLUMN "platform_svc"."fanqie_bindings"."last_sync_at" IS '最后一次与番茄平台同步的时间';
COMMENT ON COLUMN "platform_svc"."fanqie_bindings"."sync_status" IS '同步状态: pending, success, failed';
COMMENT ON COLUMN "platform_svc"."fanqie_bindings"."sync_error" IS '同步失败时的错误信息';
COMMENT ON COLUMN "platform_svc"."fanqie_bindings"."created_at" IS '绑定时间';
COMMENT ON COLUMN "platform_svc"."fanqie_bindings"."updated_at" IS '更新时间';
COMMENT ON TABLE "platform_svc"."fanqie_bindings" IS '番茄小说用户绑定表，存储 Cookie 和 CSRF Token（加密）';

-- ----------------------------
-- Records of fanqie_bindings
-- ----------------------------

-- ----------------------------
-- Table structure for platforms
-- ----------------------------
DROP TABLE IF EXISTS "platform_svc"."platforms";
CREATE TABLE "platform_svc"."platforms" (
  "id" int2 NOT NULL,
  "code" varchar(50) COLLATE "pg_catalog"."default" NOT NULL,
  "name" varchar(100) COLLATE "pg_catalog"."default" NOT NULL,
  "name_en" varchar(100) COLLATE "pg_catalog"."default",
  "icon" varchar(50) COLLATE "pg_catalog"."default",
  "auth_type" varchar(50) COLLATE "pg_catalog"."default" NOT NULL,
  "config_schema" jsonb,
  "is_active" bool NOT NULL DEFAULT true,
  "created_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
COMMENT ON COLUMN "platform_svc"."platforms"."id" IS '自增主键';
COMMENT ON COLUMN "platform_svc"."platforms"."code" IS '平台唯一编码: fanqie, qidian, qimao, zongheng, zhihu 等';
COMMENT ON COLUMN "platform_svc"."platforms"."name" IS '平台中文名称: 番茄小说, 起点中文网, 七猫小说 等';
COMMENT ON COLUMN "platform_svc"."platforms"."name_en" IS '平台英文名称: Fanqie, Qidian, Qimao 等';
COMMENT ON COLUMN "platform_svc"."platforms"."icon" IS '平台图标标识，用于前端展示';
COMMENT ON COLUMN "platform_svc"."platforms"."auth_type" IS '认证类型: cookie, oauth2, token 等';
COMMENT ON COLUMN "platform_svc"."platforms"."config_schema" IS 'JSON Schema，定义该平台绑定所需的字段结构';
COMMENT ON COLUMN "platform_svc"."platforms"."is_active" IS '平台是否启用: true=可用, false=已停用';
COMMENT ON COLUMN "platform_svc"."platforms"."created_at" IS '记录创建时间';
COMMENT ON COLUMN "platform_svc"."platforms"."updated_at" IS '记录更新时间';
COMMENT ON TABLE "platform_svc"."platforms" IS '平台基础信息表，存储各小说平台的基本配置';

-- ----------------------------
-- Records of platforms
-- ----------------------------
INSERT INTO "platform_svc"."platforms" VALUES (1, 'fanqie', '番茄小说', 'Fanqie', '🍅', 'cookie', '{"fields": ["cookie", "csrf_token"]}', 't', '2026-08-27 18:20:28.839985', '2026-08-27 18:20:28.839985');
INSERT INTO "platform_svc"."platforms" VALUES (2, 'qidian', '起点中文网', 'Qidian', '📚', 'oauth2', '{"fields": ["access_token", "refresh_token"]}', 't', '2026-08-27 18:20:28.839985', '2026-08-27 18:20:28.839985');
INSERT INTO "platform_svc"."platforms" VALUES (3, 'qimao', '七猫小说', 'Qimao', '🐱', 'cookie', '{"fields": ["cookie", "token"]}', 't', '2026-08-27 18:20:28.839985', '2026-08-27 18:20:28.839985');
INSERT INTO "platform_svc"."platforms" VALUES (4, 'zongheng', '纵横中文网', 'Zongheng', '⚔️', 'cookie', '{"fields": ["cookie"]}', 't', '2026-08-27 18:20:28.839985', '2026-08-27 18:20:28.839985');
INSERT INTO "platform_svc"."platforms" VALUES (5, 'zhihu', '知乎', 'Zhihu', '💬', 'oauth2', '{"fields": ["access_token"]}', 't', '2026-08-27 18:20:28.839985', '2026-08-27 18:20:28.839985');

-- ----------------------------
-- Table structure for publish_records
-- ----------------------------
DROP TABLE IF EXISTS "platform_svc"."publish_records";
CREATE TABLE "platform_svc"."publish_records" (
  "id" int8 NOT NULL DEFAULT nextval('"platform_svc".publish_records_id_seq'::regclass),
  "user_uuid" uuid NOT NULL,
  "platform" varchar(50) COLLATE "pg_catalog"."default" NOT NULL,
  "novel_uuid" uuid NOT NULL,
  "chapter_uuid" uuid NOT NULL,
  "chapter_title" varchar(255) COLLATE "pg_catalog"."default",
  "platform_novel_id" varchar(255) COLLATE "pg_catalog"."default",
  "platform_chapter_id" varchar(255) COLLATE "pg_catalog"."default",
  "status" varchar(50) COLLATE "pg_catalog"."default" NOT NULL,
  "error_msg" text COLLATE "pg_catalog"."default",
  "scheduled_at" timestamp(6),
  "published_at" timestamp(6),
  "retry_count" int4 DEFAULT 0,
  "created_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
COMMENT ON COLUMN "platform_svc"."publish_records"."id" IS '自增主键';
COMMENT ON COLUMN "platform_svc"."publish_records"."user_uuid" IS '发布者用户 UUID';
COMMENT ON COLUMN "platform_svc"."publish_records"."platform" IS '目标平台: fanqie, qidian, qimao 等';
COMMENT ON COLUMN "platform_svc"."publish_records"."novel_uuid" IS '关联 novel_svc.novels.uuid';
COMMENT ON COLUMN "platform_svc"."publish_records"."chapter_uuid" IS '关联 novel_svc.chapters.uuid';
COMMENT ON COLUMN "platform_svc"."publish_records"."chapter_title" IS '发布时的章节标题，快照记录';
COMMENT ON COLUMN "platform_svc"."publish_records"."platform_novel_id" IS '第三方平台的书籍ID';
COMMENT ON COLUMN "platform_svc"."publish_records"."platform_chapter_id" IS '第三方平台的章节ID';
COMMENT ON COLUMN "platform_svc"."publish_records"."status" IS '发布状态: pending, success, failed, scheduled';
COMMENT ON COLUMN "platform_svc"."publish_records"."error_msg" IS '失败时的错误信息';
COMMENT ON COLUMN "platform_svc"."publish_records"."scheduled_at" IS '预约发布时间';
COMMENT ON COLUMN "platform_svc"."publish_records"."published_at" IS '实际发布时间';
COMMENT ON COLUMN "platform_svc"."publish_records"."retry_count" IS '重试次数';
COMMENT ON COLUMN "platform_svc"."publish_records"."created_at" IS '记录创建时间';
COMMENT ON COLUMN "platform_svc"."publish_records"."updated_at" IS '记录更新时间';
COMMENT ON TABLE "platform_svc"."publish_records" IS '全平台发布历史记录表，记录每次发布操作的状态和结果';

-- ----------------------------
-- Records of publish_records
-- ----------------------------

-- ----------------------------
-- Table structure for qidian_bindings
-- ----------------------------
DROP TABLE IF EXISTS "platform_svc"."qidian_bindings";
CREATE TABLE "platform_svc"."qidian_bindings" (
  "id" int8 NOT NULL DEFAULT nextval('"platform_svc".qidian_bindings_id_seq'::regclass),
  "user_uuid" uuid NOT NULL,
  "platform_user_id" varchar(255) COLLATE "pg_catalog"."default",
  "access_token" text COLLATE "pg_catalog"."default" NOT NULL,
  "refresh_token" text COLLATE "pg_catalog"."default",
  "token_type" varchar(50) COLLATE "pg_catalog"."default" DEFAULT 'Bearer'::character varying,
  "expires_in" int4,
  "expires_at" timestamp(6),
  "scope" varchar(255) COLLATE "pg_catalog"."default",
  "is_active" bool NOT NULL DEFAULT true,
  "last_sync_at" timestamp(6),
  "sync_status" varchar(50) COLLATE "pg_catalog"."default" DEFAULT 'pending'::character varying,
  "sync_error" text COLLATE "pg_catalog"."default",
  "created_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
COMMENT ON COLUMN "platform_svc"."qidian_bindings"."id" IS '自增主键';
COMMENT ON COLUMN "platform_svc"."qidian_bindings"."user_uuid" IS '关联 user_svc.users.uuid';
COMMENT ON COLUMN "platform_svc"."qidian_bindings"."platform_user_id" IS '起点作者ID';
COMMENT ON COLUMN "platform_svc"."qidian_bindings"."access_token" IS 'OAuth2 Access Token，**需加密存储**';
COMMENT ON COLUMN "platform_svc"."qidian_bindings"."refresh_token" IS 'OAuth2 Refresh Token，用于刷新 access_token，**需加密存储**';
COMMENT ON COLUMN "platform_svc"."qidian_bindings"."token_type" IS 'Token 类型，通常为 Bearer';
COMMENT ON COLUMN "platform_svc"."qidian_bindings"."expires_in" IS 'Access Token 有效期（秒）';
COMMENT ON COLUMN "platform_svc"."qidian_bindings"."expires_at" IS 'Token 过期时间';
COMMENT ON COLUMN "platform_svc"."qidian_bindings"."scope" IS 'OAuth2 授权范围';
COMMENT ON COLUMN "platform_svc"."qidian_bindings"."is_active" IS 'Token 是否可用';
COMMENT ON COLUMN "platform_svc"."qidian_bindings"."last_sync_at" IS '最后同步时间';
COMMENT ON COLUMN "platform_svc"."qidian_bindings"."sync_status" IS '同步状态: pending, success, failed';
COMMENT ON COLUMN "platform_svc"."qidian_bindings"."sync_error" IS '同步失败错误信息';
COMMENT ON COLUMN "platform_svc"."qidian_bindings"."created_at" IS '绑定时间';
COMMENT ON COLUMN "platform_svc"."qidian_bindings"."updated_at" IS '更新时间';
COMMENT ON TABLE "platform_svc"."qidian_bindings" IS '起点中文网用户绑定表，存储 OAuth2 Token（加密）';

-- ----------------------------
-- Records of qidian_bindings
-- ----------------------------

-- ----------------------------
-- Table structure for qimao_bindings
-- ----------------------------
DROP TABLE IF EXISTS "platform_svc"."qimao_bindings";
CREATE TABLE "platform_svc"."qimao_bindings" (
  "id" int8 NOT NULL DEFAULT nextval('"platform_svc".qimao_bindings_id_seq'::regclass),
  "user_uuid" uuid NOT NULL,
  "platform_user_id" varchar(255) COLLATE "pg_catalog"."default",
  "cookie" text COLLATE "pg_catalog"."default" NOT NULL,
  "token" varchar(500) COLLATE "pg_catalog"."default",
  "expires_at" timestamp(6),
  "is_active" bool NOT NULL DEFAULT true,
  "last_sync_at" timestamp(6),
  "sync_status" varchar(50) COLLATE "pg_catalog"."default" DEFAULT 'pending'::character varying,
  "sync_error" text COLLATE "pg_catalog"."default",
  "created_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
COMMENT ON COLUMN "platform_svc"."qimao_bindings"."id" IS '自增主键';
COMMENT ON COLUMN "platform_svc"."qimao_bindings"."user_uuid" IS '关联 user_svc.users.uuid';
COMMENT ON COLUMN "platform_svc"."qimao_bindings"."platform_user_id" IS '七猫作者ID';
COMMENT ON COLUMN "platform_svc"."qimao_bindings"."cookie" IS '七猫作家后台 Cookie，**需 AES-256-GCM 加密存储**';
COMMENT ON COLUMN "platform_svc"."qimao_bindings"."token" IS '额外的 Token 或鉴权凭证，**需加密存储**';
COMMENT ON COLUMN "platform_svc"."qimao_bindings"."expires_at" IS '凭证过期时间';
COMMENT ON COLUMN "platform_svc"."qimao_bindings"."is_active" IS '凭证是否可用';
COMMENT ON COLUMN "platform_svc"."qimao_bindings"."last_sync_at" IS '最后同步时间';
COMMENT ON COLUMN "platform_svc"."qimao_bindings"."sync_status" IS '同步状态: pending, success, failed';
COMMENT ON COLUMN "platform_svc"."qimao_bindings"."sync_error" IS '同步失败错误信息';
COMMENT ON COLUMN "platform_svc"."qimao_bindings"."created_at" IS '绑定时间';
COMMENT ON COLUMN "platform_svc"."qimao_bindings"."updated_at" IS '更新时间';
COMMENT ON TABLE "platform_svc"."qimao_bindings" IS '七猫小说用户绑定表，存储 Cookie 和 Token（加密）';

-- ----------------------------
-- Records of qimao_bindings
-- ----------------------------

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
ALTER SEQUENCE "platform_svc"."fanqie_bindings_id_seq"
OWNED BY "platform_svc"."fanqie_bindings"."id";
SELECT setval('"platform_svc"."fanqie_bindings_id_seq"', 1, false);

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
ALTER SEQUENCE "platform_svc"."publish_records_id_seq"
OWNED BY "platform_svc"."publish_records"."id";
SELECT setval('"platform_svc"."publish_records_id_seq"', 1, false);

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
ALTER SEQUENCE "platform_svc"."qidian_bindings_id_seq"
OWNED BY "platform_svc"."qidian_bindings"."id";
SELECT setval('"platform_svc"."qidian_bindings_id_seq"', 1, false);

-- ----------------------------
-- Alter sequences owned by
-- ----------------------------
ALTER SEQUENCE "platform_svc"."qimao_bindings_id_seq"
OWNED BY "platform_svc"."qimao_bindings"."id";
SELECT setval('"platform_svc"."qimao_bindings_id_seq"', 1, false);

-- ----------------------------
-- Indexes structure for table fanqie_bindings
-- ----------------------------
CREATE INDEX "idx_fanqie_active" ON "platform_svc"."fanqie_bindings" USING btree (
  "is_active" "pg_catalog"."bool_ops" ASC NULLS LAST
);
CREATE INDEX "idx_fanqie_expires" ON "platform_svc"."fanqie_bindings" USING btree (
  "expires_at" "pg_catalog"."timestamp_ops" ASC NULLS LAST
);
CREATE INDEX "idx_fanqie_user" ON "platform_svc"."fanqie_bindings" USING btree (
  "user_uuid" "pg_catalog"."uuid_ops" ASC NULLS LAST
);

-- ----------------------------
-- Uniques structure for table fanqie_bindings
-- ----------------------------
ALTER TABLE "platform_svc"."fanqie_bindings" ADD CONSTRAINT "uk_fanqie_user" UNIQUE ("user_uuid");

-- ----------------------------
-- Primary Key structure for table fanqie_bindings
-- ----------------------------
ALTER TABLE "platform_svc"."fanqie_bindings" ADD CONSTRAINT "fanqie_bindings_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Uniques structure for table platforms
-- ----------------------------
ALTER TABLE "platform_svc"."platforms" ADD CONSTRAINT "uk_platforms_code" UNIQUE ("code");

-- ----------------------------
-- Primary Key structure for table platforms
-- ----------------------------
ALTER TABLE "platform_svc"."platforms" ADD CONSTRAINT "platforms_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table publish_records
-- ----------------------------
CREATE INDEX "idx_publish_novel" ON "platform_svc"."publish_records" USING btree (
  "novel_uuid" "pg_catalog"."uuid_ops" ASC NULLS LAST
);
CREATE INDEX "idx_publish_platform" ON "platform_svc"."publish_records" USING btree (
  "platform" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE INDEX "idx_publish_scheduled" ON "platform_svc"."publish_records" USING btree (
  "scheduled_at" "pg_catalog"."timestamp_ops" ASC NULLS LAST
) WHERE status::text = 'scheduled'::text;
CREATE INDEX "idx_publish_status" ON "platform_svc"."publish_records" USING btree (
  "status" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE INDEX "idx_publish_user" ON "platform_svc"."publish_records" USING btree (
  "user_uuid" "pg_catalog"."uuid_ops" ASC NULLS LAST
);

-- ----------------------------
-- Primary Key structure for table publish_records
-- ----------------------------
ALTER TABLE "platform_svc"."publish_records" ADD CONSTRAINT "publish_records_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table qidian_bindings
-- ----------------------------
CREATE INDEX "idx_qidian_active" ON "platform_svc"."qidian_bindings" USING btree (
  "is_active" "pg_catalog"."bool_ops" ASC NULLS LAST
);
CREATE INDEX "idx_qidian_user" ON "platform_svc"."qidian_bindings" USING btree (
  "user_uuid" "pg_catalog"."uuid_ops" ASC NULLS LAST
);

-- ----------------------------
-- Uniques structure for table qidian_bindings
-- ----------------------------
ALTER TABLE "platform_svc"."qidian_bindings" ADD CONSTRAINT "uk_qidian_user" UNIQUE ("user_uuid");

-- ----------------------------
-- Primary Key structure for table qidian_bindings
-- ----------------------------
ALTER TABLE "platform_svc"."qidian_bindings" ADD CONSTRAINT "qidian_bindings_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table qimao_bindings
-- ----------------------------
CREATE INDEX "idx_qimao_active" ON "platform_svc"."qimao_bindings" USING btree (
  "is_active" "pg_catalog"."bool_ops" ASC NULLS LAST
);
CREATE INDEX "idx_qimao_user" ON "platform_svc"."qimao_bindings" USING btree (
  "user_uuid" "pg_catalog"."uuid_ops" ASC NULLS LAST
);

-- ----------------------------
-- Uniques structure for table qimao_bindings
-- ----------------------------
ALTER TABLE "platform_svc"."qimao_bindings" ADD CONSTRAINT "uk_qimao_user" UNIQUE ("user_uuid");

-- ----------------------------
-- Primary Key structure for table qimao_bindings
-- ----------------------------
ALTER TABLE "platform_svc"."qimao_bindings" ADD CONSTRAINT "qimao_bindings_pkey" PRIMARY KEY ("id");

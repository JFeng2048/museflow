-- ============================================================================
-- config_svc —— 系统配置域
--
-- 归属服务：services/config-service（gRPC，端口 5004；5003 已被 crawl4ai-service 占用）
--
-- 与 user_svc.sql / platform_svc.sql 完全同一套约定：
--   * int8 自增主键 + 每表独立序列；
--   * created_at / updated_at 由 DEFAULT CURRENT_TIMESTAMP 初始化，
--     updated_at 另配 BEFORE UPDATE 触发器自动刷新（触发器函数见本文件末尾）；
--   * 全部字段中文 COMMENT；
--   * 序列按 max(id) setval 对齐，避免显式 id 种子插入后 nextval 撞主键
--     （SQLSTATE 23505，且失败同样消耗序号，表现为「第一次失败重试就好」）。
--     写法固定为 setval(seq, COALESCE(max(id), 0) + 1, false)：空表得 1、非空表得
--     max(id)+1。绝不能用 COALESCE(max(id), 0) 配 true，空表会传 0 给 setval 而越界；
--   * 跨域只做逻辑关联、不建物理外键：user_uuid 关联 user_svc.user.uuid，
--     schema 内的 provider_id / user_provider_id 也不加 FOREIGN KEY，
--     便于后续拆库与灰度迁移。
--   * COLLATE "pg_catalog"."default" 只允许出现在 varchar / text / char 列上。
--     本套 DDL 由 Navicat 导出风格演化而来，导出器会给文本列统一带上 COLLATE，
--     但 PostgreSQL 对 int8 / int4 / uuid / bool / timestamp 等非字符类型
--     不接受排序规则，写了会直接报
--       ERROR: collations are not supported by type bigint
--     因此数值、UUID、布尔、时间列一律不带 COLLATE，新增列时不要照抄。
--
-- 加密约定（务必遵守）：
--   api_key / secret_value 一律存放 AES-256-GCM 密文，格式
--       v1:<nonce_b64>:<ciphertext_b64>
--   密钥取自共享环境变量 MODEL_SECRET_KEY（无前缀，与 JWT_SECRET / DB_* 同级），
--   由 config-service 在写入前加密、读取时解密。
--   绝不明文落库，绝不下发前端：对外只回 api_key_hint（末 4 位）与
--   api_key_updated_at；LLM 调用一律在服务端完成，前端永远拿不到 base_url + key。
--
-- 【警告】本文件与 user_svc.sql 同为「DROP + CREATE」全量脚本，用于空库初始化。
--   在已有数据的库上重复执行会清空 config_svc 全部表；增量变更请改走
--   database/migrations/ 下的编号迁移脚本。
-- ============================================================================

-- ----------------------------
-- Schema structure for config_svc
-- ----------------------------
-- 注：user_svc.sql / platform_svc.sql 均未包含建库语句（由 DBA 手工创建），
-- 这里显式补上，保证新环境按本文件即可初始化，不必依赖外部步骤。
DROP SCHEMA IF EXISTS "config_svc" CASCADE;
CREATE SCHEMA "config_svc";
COMMENT ON SCHEMA "config_svc" IS '系统配置域：模型渠道、模型目录、用户自定义模型与通用系统配置';

-- ----------------------------
-- Sequence structure for model_provider_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "config_svc"."model_provider_id_seq";
CREATE SEQUENCE "config_svc"."model_provider_id_seq"
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;

-- ----------------------------
-- Sequence structure for model_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "config_svc"."model_id_seq";
CREATE SEQUENCE "config_svc"."model_id_seq"
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;

-- ----------------------------
-- Sequence structure for user_model_provider_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "config_svc"."user_model_provider_id_seq";
CREATE SEQUENCE "config_svc"."user_model_provider_id_seq"
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;

-- ----------------------------
-- Sequence structure for user_model_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "config_svc"."user_model_id_seq";
CREATE SEQUENCE "config_svc"."user_model_id_seq"
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;

-- ----------------------------
-- Sequence structure for system_setting_id_seq
-- ----------------------------
DROP SEQUENCE IF EXISTS "config_svc"."system_setting_id_seq";
CREATE SEQUENCE "config_svc"."system_setting_id_seq"
INCREMENT 1
MINVALUE  1
MAXVALUE 9223372036854775807
START 1
CACHE 1;

-- ----------------------------
-- Table structure for model_provider
-- ----------------------------
-- 平台侧模型渠道：厂商 + base_url + api_key，由管理端维护，是「平台提供的」那一半。
-- 一行 = 一个可用的上游渠道；模型明细挂在其下（config_svc.model）。
DROP TABLE IF EXISTS "config_svc"."model_provider";
CREATE TABLE "config_svc"."model_provider" (
  "id" int8 NOT NULL DEFAULT nextval('"config_svc".model_provider_id_seq'::regclass),
  "code" varchar(50) COLLATE "pg_catalog"."default" NOT NULL,
  "name" varchar(100) COLLATE "pg_catalog"."default" NOT NULL,
  "protocol" varchar(50) COLLATE "pg_catalog"."default" NOT NULL,
  "base_url" varchar(500) COLLATE "pg_catalog"."default" NOT NULL,
  "api_key" text COLLATE "pg_catalog"."default" NOT NULL,
  "api_key_hint" varchar(8) COLLATE "pg_catalog"."default",
  "api_key_updated_at" timestamp(6),
  "organization" varchar(255) COLLATE "pg_catalog"."default",
  "extra" jsonb,
  "is_active" bool NOT NULL DEFAULT true,
  "sort_order" int4 NOT NULL DEFAULT 0,
  "created_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
COMMENT ON COLUMN "config_svc"."model_provider"."id" IS '自增主键，仅内部使用';
COMMENT ON COLUMN "config_svc"."model_provider"."code" IS '渠道唯一编码，供业务代码引用: openai/anthropic/gemini/zhipu/deepseek/qwen/museflow 等';
COMMENT ON COLUMN "config_svc"."model_provider"."name" IS '展示名: OpenAI / Anthropic / 智谱 / MuseFlow 官方';
COMMENT ON COLUMN "config_svc"."model_provider"."protocol" IS '通讯协议: openai(OpenAI 兼容)/anthropic/gemini/custom';
COMMENT ON COLUMN "config_svc"."model_provider"."base_url" IS 'API 基础地址，如 https://api.openai.com/v1';
COMMENT ON COLUMN "config_svc"."model_provider"."api_key" IS '平台 API Key，AES-256-GCM 密文(v1:nonce:ciphertext)，只写不读';
COMMENT ON COLUMN "config_svc"."model_provider"."api_key_hint" IS 'API Key 末 4 位，仅用于后台脱敏展示与轮换确认';
COMMENT ON COLUMN "config_svc"."model_provider"."api_key_updated_at" IS 'Key 最后一次轮换时间，与 updated_at 区分：后者任何字段变更都会动';
COMMENT ON COLUMN "config_svc"."model_provider"."organization" IS '组织标识，部分厂商(如 OpenAI)需要，可空';
COMMENT ON COLUMN "config_svc"."model_provider"."extra" IS '协议特有扩展，JSONB 格式: {"api_version":"2024-02-01","region":"cn"}';
COMMENT ON COLUMN "config_svc"."model_provider"."is_active" IS '渠道是否启用: true=可用, false=已停用(停用后其下模型一律不可见)';
COMMENT ON COLUMN "config_svc"."model_provider"."sort_order" IS '后台展示排序，值越小越靠前';
COMMENT ON COLUMN "config_svc"."model_provider"."created_at" IS '记录创建时间';
COMMENT ON COLUMN "config_svc"."model_provider"."updated_at" IS '记录更新时间，由触发器维护';
COMMENT ON TABLE "config_svc"."model_provider" IS '平台模型渠道表，存储厂商 base_url 与 api_key(加密)，由管理端维护';

-- ----------------------------
-- Table structure for model
-- ----------------------------
-- 平台侧模型明细：挂在某个平台渠道之下，是用户端「刷出来」的可用模型主体。
DROP TABLE IF EXISTS "config_svc"."model";
CREATE TABLE "config_svc"."model" (
  "id" int8 NOT NULL DEFAULT nextval('"config_svc".model_id_seq'::regclass),
  "provider_id" int8 NOT NULL,
  "code" varchar(100) COLLATE "pg_catalog"."default" NOT NULL,
  "name" varchar(100) COLLATE "pg_catalog"."default" NOT NULL,
  "model_type" varchar(50) COLLATE "pg_catalog"."default" NOT NULL,
  "api_model" varchar(100) COLLATE "pg_catalog"."default" NOT NULL,
  "context_window" int4 NOT NULL DEFAULT 0,
  "max_output_tokens" int4,
  "capabilities" jsonb,
  "credit_cost" int4 NOT NULL DEFAULT 0,
  "description" text COLLATE "pg_catalog"."default",
  "is_active" bool NOT NULL DEFAULT true,
  "sort_order" int4 NOT NULL DEFAULT 0,
  "created_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
COMMENT ON COLUMN "config_svc"."model"."id" IS '自增主键，仅内部使用';
COMMENT ON COLUMN "config_svc"."model"."provider_id" IS '关联 config_svc.model_provider.id(逻辑关联，无物理外键)';
COMMENT ON COLUMN "config_svc"."model"."code" IS '模型唯一编码，供业务代码引用，如 muse-pro/muse-embed-v1';
COMMENT ON COLUMN "config_svc"."model"."name" IS '展示名: MusePro 创作主力 / GPT-4o';
COMMENT ON COLUMN "config_svc"."model"."model_type" IS '模型类型: chat(对话大模型)/embedding(向量化)/rerank(重排序)/vision(视觉理解)/image(图像生成)/audio(语音)/video(视频)';
COMMENT ON COLUMN "config_svc"."model"."api_model" IS '实际调用时传给上游的模型标识，如 gpt-4o、muse-pro';
COMMENT ON COLUMN "config_svc"."model"."context_window" IS '上下文窗口上限(token)，0 表示未知或不限制';
COMMENT ON COLUMN "config_svc"."model"."max_output_tokens" IS '单次响应 token 上限，可空表示跟随上游默认';
COMMENT ON COLUMN "config_svc"."model"."capabilities" IS '能力开关，JSONB 格式: {"stream":true,"tool_call":true,"json_mode":false,"vision":false}';
COMMENT ON COLUMN "config_svc"."model"."credit_cost" IS '单次调用消耗积分(平台计费)，0 表示免费';
COMMENT ON COLUMN "config_svc"."model"."description" IS '模型用途说明，展示在用户端模型选择器';
COMMENT ON COLUMN "config_svc"."model"."is_active" IS '是否对用户可见: true=可用, false=已下架';
COMMENT ON COLUMN "config_svc"."model"."sort_order" IS '后台与用户端展示排序，值越小越靠前';
COMMENT ON COLUMN "config_svc"."model"."created_at" IS '记录创建时间';
COMMENT ON COLUMN "config_svc"."model"."updated_at" IS '记录更新时间，由触发器维护';
COMMENT ON TABLE "config_svc"."model" IS '平台模型明细表，定义模型类型、调用标识、上下文窗口与计费';

-- ----------------------------
-- Table structure for user_model_provider
-- ----------------------------
-- 用户自定义渠道：base_url / api_key 由用户本人填写，用 user_uuid 做逻辑关联。
-- 与 model_provider 的区别：没有 code(不被业务代码引用)、没有 sort_order(用户自有排序)。
DROP TABLE IF EXISTS "config_svc"."user_model_provider";
CREATE TABLE "config_svc"."user_model_provider" (
  "id" int8 NOT NULL DEFAULT nextval('"config_svc".user_model_provider_id_seq'::regclass),
  "user_uuid" uuid NOT NULL,
  "name" varchar(100) COLLATE "pg_catalog"."default" NOT NULL,
  "protocol" varchar(50) COLLATE "pg_catalog"."default" NOT NULL,
  "base_url" varchar(500) COLLATE "pg_catalog"."default" NOT NULL,
  "api_key" text COLLATE "pg_catalog"."default" NOT NULL,
  "api_key_hint" varchar(8) COLLATE "pg_catalog"."default",
  "api_key_updated_at" timestamp(6),
  "organization" varchar(255) COLLATE "pg_catalog"."default",
  "extra" jsonb,
  "is_active" bool NOT NULL DEFAULT true,
  "created_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
COMMENT ON COLUMN "config_svc"."user_model_provider"."id" IS '自增主键，仅内部使用';
COMMENT ON COLUMN "config_svc"."user_model_provider"."user_uuid" IS '关联 user_svc.user.uuid(逻辑关联，无物理外键)';
COMMENT ON COLUMN "config_svc"."user_model_provider"."name" IS '用户自定义渠道名，同一用户下唯一';
COMMENT ON COLUMN "config_svc"."user_model_provider"."protocol" IS '通讯协议: openai(OpenAI 兼容)/anthropic/gemini/custom';
COMMENT ON COLUMN "config_svc"."user_model_provider"."base_url" IS '用户自填的 API 基础地址';
COMMENT ON COLUMN "config_svc"."user_model_provider"."api_key" IS '用户 API Key，AES-256-GCM 密文(v1:nonce:ciphertext)';
COMMENT ON COLUMN "config_svc"."user_model_provider"."api_key_hint" IS 'API Key 末 4 位，创建后可查看一次，此后只展示该值';
COMMENT ON COLUMN "config_svc"."user_model_provider"."api_key_updated_at" IS 'Key 最后一次覆盖时间';
COMMENT ON COLUMN "config_svc"."user_model_provider"."organization" IS '组织标识，部分厂商需要，可空';
COMMENT ON COLUMN "config_svc"."user_model_provider"."extra" IS '协议特有扩展，JSONB 格式';
COMMENT ON COLUMN "config_svc"."user_model_provider"."is_active" IS '是否启用: true=可用, false=已停用(可自行恢复)';
COMMENT ON COLUMN "config_svc"."user_model_provider"."created_at" IS '记录创建时间';
COMMENT ON COLUMN "config_svc"."user_model_provider"."updated_at" IS '记录更新时间，由触发器维护';
COMMENT ON TABLE "config_svc"."user_model_provider" IS '用户自定义模型渠道表，用户自带 base_url 与 api_key(加密)';

-- ----------------------------
-- Table structure for user_model
-- ----------------------------
-- 用户自选模型：只能挂在用户自己的渠道下，不计平台积分(用户直接向上游付费)。
-- 没有 credit_cost 字段，即约定自定义模型对平台免费。
DROP TABLE IF EXISTS "config_svc"."user_model";
CREATE TABLE "config_svc"."user_model" (
  "id" int8 NOT NULL DEFAULT nextval('"config_svc".user_model_id_seq'::regclass),
  "user_uuid" uuid NOT NULL,
  "user_provider_id" int8 NOT NULL,
  "name" varchar(100) COLLATE "pg_catalog"."default" NOT NULL,
  "model_type" varchar(50) COLLATE "pg_catalog"."default" NOT NULL,
  "api_model" varchar(100) COLLATE "pg_catalog"."default" NOT NULL,
  "context_window" int4 NOT NULL DEFAULT 0,
  "max_output_tokens" int4,
  "capabilities" jsonb,
  "description" text COLLATE "pg_catalog"."default",
  "is_active" bool NOT NULL DEFAULT true,
  "created_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP
)
;
COMMENT ON COLUMN "config_svc"."user_model"."id" IS '自增主键，仅内部使用';
COMMENT ON COLUMN "config_svc"."user_model"."user_uuid" IS '关联 user_svc.user.uuid(逻辑关联，无物理外键)';
COMMENT ON COLUMN "config_svc"."user_model"."user_provider_id" IS '关联 config_svc.user_model_provider.id(逻辑关联，无物理外键)';
COMMENT ON COLUMN "config_svc"."user_model"."name" IS '展示名: 我的 GPT-4o / 本地 Qwen';
COMMENT ON COLUMN "config_svc"."user_model"."model_type" IS '模型类型，取值同 config_svc.model.model_type';
COMMENT ON COLUMN "config_svc"."user_model"."api_model" IS '实际调用时传给上游的模型标识';
COMMENT ON COLUMN "config_svc"."user_model"."context_window" IS '上下文窗口上限(token)，0 表示未知或不限制';
COMMENT ON COLUMN "config_svc"."user_model"."max_output_tokens" IS '单次响应 token 上限，可空表示跟随上游默认';
COMMENT ON COLUMN "config_svc"."user_model"."capabilities" IS '能力开关，JSONB 格式: {"stream":true,"tool_call":false,"json_mode":false,"vision":false}';
COMMENT ON COLUMN "config_svc"."user_model"."description" IS '备注，仅用户本人可见';
COMMENT ON COLUMN "config_svc"."user_model"."is_active" IS '是否启用: true=可用, false=已停用';
COMMENT ON COLUMN "config_svc"."user_model"."created_at" IS '记录创建时间';
COMMENT ON COLUMN "config_svc"."user_model"."updated_at" IS '记录更新时间，由触发器维护';
COMMENT ON TABLE "config_svc"."user_model" IS '用户自选模型表，挂在用户自定义渠道下，不计平台积分';

-- ----------------------------
-- Table structure for system_setting
-- ----------------------------
-- 通用系统配置载体：key-value + jsonb，承接「其他系统可变更的配置」。
-- 非机密值放 value，机密值(令牌/密码/第三方 secret)放 secret_value(密文)，
-- 两者由 CHECK 约束保证恰好只有一个非空，避免同一份配置存两处。
DROP TABLE IF EXISTS "config_svc"."system_setting";
CREATE TABLE "config_svc"."system_setting" (
  "id" int8 NOT NULL DEFAULT nextval('"config_svc".system_setting_id_seq'::regclass),
  "config_group" varchar(50) COLLATE "pg_catalog"."default" NOT NULL,
  "key" varchar(100) COLLATE "pg_catalog"."default" NOT NULL,
  "value" jsonb,
  "secret_value" text COLLATE "pg_catalog"."default",
  "is_secret" bool NOT NULL DEFAULT false,
  "value_type" varchar(20) COLLATE "pg_catalog"."default" NOT NULL DEFAULT 'json'::character varying,
  "is_public" bool NOT NULL DEFAULT false,
  "description" text COLLATE "pg_catalog"."default",
  "created_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT "ck_system_setting_value" CHECK (("value" IS NULL) <> ("secret_value" IS NULL)),
  CONSTRAINT "ck_system_setting_secret" CHECK ("is_secret" = ("secret_value" IS NOT NULL))
)
;
COMMENT ON COLUMN "config_svc"."system_setting"."id" IS '自增主键，仅内部使用';
COMMENT ON COLUMN "config_svc"."system_setting"."config_group" IS '配置分组: model/general/notify/quota 等，同组内 key 唯一';
COMMENT ON COLUMN "config_svc"."system_setting"."key" IS '配置键，如 default_chat_model / site_announcement';
COMMENT ON COLUMN "config_svc"."system_setting"."value" IS '非机密配置值，任意 JSON；is_secret=true 时必须为 NULL';
COMMENT ON COLUMN "config_svc"."system_setting"."secret_value" IS '机密配置值，AES-256-GCM 密文(v1:nonce:ciphertext)；is_secret=false 时必须为 NULL';
COMMENT ON COLUMN "config_svc"."system_setting"."is_secret" IS '是否为机密配置，由 CHECK 约束与 secret_value 是否非空保持一致';
COMMENT ON COLUMN "config_svc"."system_setting"."value_type" IS '值类型提示，用于后台表单渲染: string/number/bool/json';
COMMENT ON COLUMN "config_svc"."system_setting"."is_public" IS '是否可下发前端: true=可随公开配置接口返回, false=仅后台可见';
COMMENT ON COLUMN "config_svc"."system_setting"."description" IS '配置项说明，展示在后台表单';
COMMENT ON COLUMN "config_svc"."system_setting"."created_at" IS '记录创建时间';
COMMENT ON COLUMN "config_svc"."system_setting"."updated_at" IS '记录更新时间，由触发器维护';
COMMENT ON TABLE "config_svc"."system_setting" IS '通用系统配置表，key-value 形式承接可变更的系统级配置';

-- ----------------------------
-- Function structure for trigger_set_updated_at
-- ----------------------------
DROP FUNCTION IF EXISTS "config_svc"."trigger_set_updated_at"();
CREATE FUNCTION "config_svc"."trigger_set_updated_at"()
  RETURNS "pg_catalog"."trigger" AS $BODY$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$BODY$
  LANGUAGE plpgsql VOLATILE
  COST 100;
COMMENT ON FUNCTION "config_svc"."trigger_set_updated_at"() IS '自动更新updated_at字段的触发器函数';

-- ----------------------------
-- 序列对齐（必须）
-- ----------------------------
-- 与 user_svc.sql 同一理由：显式 id 插入不推进序列，必须按现有数据 max(id) 对齐，
-- 否则 INSERT 会撞主键报 duplicate key value violates unique constraint (SQLSTATE 23505)。
--    setval(seq, n, true)  -> 下一次 nextval 返回 n+1
--    setval(seq, n, false) -> 下一次 nextval 返回 n
-- 这里统一取「max(id)+1 配 is_called=false」，语义就是「下一次 nextval 正好取这个值」。
-- 关键是空表也不能传 0：序列 MINVALUE 为 1，setval 会做区间校验，传 0 直接报
--   ERROR: setval: value 0 is out of bounds for sequence "xxx" (1..9223372036854775807)
-- 所以先 COALESCE 回退到 0 再加 1，空表得 1、非空表得 max(id)+1，两种情况都合法。
-- 本文件不写种子数据（渠道与模型由管理端维护，Key 属于机密不落脚本），空库即为上述空表分支。
ALTER SEQUENCE "config_svc"."model_provider_id_seq"
OWNED BY "config_svc"."model_provider"."id";
SELECT setval('"config_svc"."model_provider_id_seq"', COALESCE((SELECT max(id) FROM "config_svc"."model_provider"), 0) + 1, false);

ALTER SEQUENCE "config_svc"."model_id_seq"
OWNED BY "config_svc"."model"."id";
SELECT setval('"config_svc"."model_id_seq"', COALESCE((SELECT max(id) FROM "config_svc"."model"), 0) + 1, false);

ALTER SEQUENCE "config_svc"."user_model_provider_id_seq"
OWNED BY "config_svc"."user_model_provider"."id";
SELECT setval('"config_svc"."user_model_provider_id_seq"', COALESCE((SELECT max(id) FROM "config_svc"."user_model_provider"), 0) + 1, false);

ALTER SEQUENCE "config_svc"."user_model_id_seq"
OWNED BY "config_svc"."user_model"."id";
SELECT setval('"config_svc"."user_model_id_seq"', COALESCE((SELECT max(id) FROM "config_svc"."user_model"), 0) + 1, false);

ALTER SEQUENCE "config_svc"."system_setting_id_seq"
OWNED BY "config_svc"."system_setting"."id";
SELECT setval('"config_svc"."system_setting_id_seq"', COALESCE((SELECT max(id) FROM "config_svc"."system_setting"), 0) + 1, false);

-- ----------------------------
-- Indexes structure for table model_provider
-- ----------------------------
CREATE UNIQUE INDEX "idx_model_provider_code" ON "config_svc"."model_provider" USING btree (
  "code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE INDEX "idx_model_provider_active" ON "config_svc"."model_provider" USING btree (
  "is_active" "pg_catalog"."bool_ops" ASC NULLS LAST,
  "sort_order" "pg_catalog"."int4_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table model_provider
-- ----------------------------
CREATE TRIGGER "trg_model_provider_updated_at" BEFORE UPDATE ON "config_svc"."model_provider"
FOR EACH ROW
EXECUTE PROCEDURE "config_svc"."trigger_set_updated_at"();

-- ----------------------------
-- Primary Key structure for table model_provider
-- ----------------------------
ALTER TABLE "config_svc"."model_provider" ADD CONSTRAINT "model_provider_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table model
-- ----------------------------
CREATE UNIQUE INDEX "idx_model_code" ON "config_svc"."model" USING btree (
  "code" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE UNIQUE INDEX "idx_model_provider_api_model" ON "config_svc"."model" USING btree (
  "provider_id" "pg_catalog"."int8_ops" ASC NULLS LAST,
  "api_model" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE INDEX "idx_model_provider" ON "config_svc"."model" USING btree (
  "provider_id" "pg_catalog"."int8_ops" ASC NULLS LAST
);
CREATE INDEX "idx_model_type" ON "config_svc"."model" USING btree (
  "model_type" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE INDEX "idx_model_active" ON "config_svc"."model" USING btree (
  "is_active" "pg_catalog"."bool_ops" ASC NULLS LAST,
  "model_type" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "sort_order" "pg_catalog"."int4_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table model
-- ----------------------------
CREATE TRIGGER "trg_model_updated_at" BEFORE UPDATE ON "config_svc"."model"
FOR EACH ROW
EXECUTE PROCEDURE "config_svc"."trigger_set_updated_at"();

-- ----------------------------
-- Primary Key structure for table model
-- ----------------------------
ALTER TABLE "config_svc"."model" ADD CONSTRAINT "model_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table user_model_provider
-- ----------------------------
CREATE UNIQUE INDEX "idx_user_provider_user_name" ON "config_svc"."user_model_provider" USING btree (
  "user_uuid" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "name" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE INDEX "idx_user_provider_user" ON "config_svc"."user_model_provider" USING btree (
  "user_uuid" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "is_active" "pg_catalog"."bool_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table user_model_provider
-- ----------------------------
CREATE TRIGGER "trg_user_model_provider_updated_at" BEFORE UPDATE ON "config_svc"."user_model_provider"
FOR EACH ROW
EXECUTE PROCEDURE "config_svc"."trigger_set_updated_at"();

-- ----------------------------
-- Primary Key structure for table user_model_provider
-- ----------------------------
ALTER TABLE "config_svc"."user_model_provider" ADD CONSTRAINT "user_model_provider_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table user_model
-- ----------------------------
CREATE UNIQUE INDEX "idx_user_model_user_api_model" ON "config_svc"."user_model" USING btree (
  "user_uuid" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "user_provider_id" "pg_catalog"."int8_ops" ASC NULLS LAST,
  "api_model" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);
CREATE INDEX "idx_user_model_user" ON "config_svc"."user_model" USING btree (
  "user_uuid" "pg_catalog"."uuid_ops" ASC NULLS LAST,
  "is_active" "pg_catalog"."bool_ops" ASC NULLS LAST
);
CREATE INDEX "idx_user_model_provider" ON "config_svc"."user_model" USING btree (
  "user_provider_id" "pg_catalog"."int8_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table user_model
-- ----------------------------
CREATE TRIGGER "trg_user_model_updated_at" BEFORE UPDATE ON "config_svc"."user_model"
FOR EACH ROW
EXECUTE PROCEDURE "config_svc"."trigger_set_updated_at"();

-- ----------------------------
-- Primary Key structure for table user_model
-- ----------------------------
ALTER TABLE "config_svc"."user_model" ADD CONSTRAINT "user_model_pkey" PRIMARY KEY ("id");

-- ----------------------------
-- Indexes structure for table system_setting
-- ----------------------------
CREATE UNIQUE INDEX "idx_system_setting_group_key" ON "config_svc"."system_setting" USING btree (
  "config_group" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST,
  "key" COLLATE "pg_catalog"."default" "pg_catalog"."text_ops" ASC NULLS LAST
);

-- ----------------------------
-- Triggers structure for table system_setting
-- ----------------------------
CREATE TRIGGER "trg_system_setting_updated_at" BEFORE UPDATE ON "config_svc"."system_setting"
FOR EACH ROW
EXECUTE PROCEDURE "config_svc"."trigger_set_updated_at"();

-- ----------------------------
-- Primary Key structure for table system_setting
-- ----------------------------
ALTER TABLE "config_svc"."system_setting" ADD CONSTRAINT "system_setting_pkey" PRIMARY KEY ("id");

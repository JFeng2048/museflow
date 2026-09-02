# museflow.com · Nginx 配置

本目录是 `museflow.com`（jfeng 的个人介绍站）在服务器上使用的 Nginx 配置，含一套**通用错误页**（给 Nginx 服务本身使用，不绑定具体项目）。

## 目录结构

```
deploy/nginx/
├── nginx.conf                 # 顶层配置（mime、日志、include conf.d）
├── conf.d/
│   └── museflow.com.conf      # 个人站点配置：静态托管 + 错误页映射
└── errors/                    # 通用 Nginx 错误页（共享 error.css）
    ├── error.css
    ├── 400.html  403.html  404.html
    └── 500.html  502.html  503.html  504.html
```

## 服务器上的部署

将文件放到对应位置（与 `museflow.com.conf` 中的路径一致）：

```bash
# 站点静态产物
/var/www/museflow.com/            # index.html 等个人站文件
/var/www/museflow.com/errors/     # 错误页（与 css 同源）

# Nginx 配置
/etc/nginx/nginx.conf             # 顶层（或直接用系统默认 + 以下）
/etc/nginx/conf.d/museflow.com.conf
```

```bash
# 校验并重载
sudo nginx -t
sudo nginx -reload
```

## 错误页说明

- 每个状态码一个独立 HTML（Nginx 推荐做法），共享 `errors/error.css`。
- 风格：中性、克制、专业，不绑定任何项目；5xx 以危险红点缀。
- 文案通用（页面未找到 / 禁止访问 / 服务器内部错误 等），带返回首页 / 重试按钮。
- 浏览器通过绝对路径 `/errors/error.css` 加载样式，由 `museflow.com.conf` 的
  `location ^~ /errors/` 提供。

## 个人介绍站内容建议

`museflow.com` 是 jfeng 的个人介绍站。可在此介绍技术栈与作品，例如把
MuseFlow（AI 小说生成平台，Go 微服务 + Vue3 前端）作为代表作展示。
站点 HTML 由你自行维护于 `/var/www/museflow.com/`，本目录只负责 Nginx 托管与错误页。

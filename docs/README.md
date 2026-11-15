# lathe-base 文档站

设计讨论稿；业务方案尚未确认、尚未实现。公开文档源位于 `content/`，不包含业务实现。站点准备采用 GitHub 项目 Pages 地址 `https://shyun-cc.github.io/lathe-base/`；这不是上线声明。

## 工具与来源

- Git
- Go **1.27.3**
- Hugo **Extended 0.168.0**（必须为 extended 构建）
- Hugo Module：`github.com/pgsty/oink v1.5.1`，锁定校验和见 `go.sum`

版本依据：[OINK README](https://github.com/pgsty/oink/blob/v1.5.1/README.md)、[官方 starter](https://github.com/pgsty/oink-starter)（核验参考提交 `6662307ed544eefcfe7bd80a15bffbf3daedc147`）。无需 Node.js、npm 或外部 Sass。首次构建需要网络下载模块。模块缓存与工具链不提交到仓库。

## 预览与构建

从仓库根目录运行：

```sh
cd docs
export GOWORK=off HUGO_MODULE_WORKSPACE=off
go mod download
go mod verify
hugo server --bind 127.0.0.1
```

按 Hugo 输出的包含 `/lathe-base/` 的本地 URL 预览。生产构建：

```sh
hugo --cleanDestinationDir --gc --minify --environment production --printPathWarnings
```

生成文件位于 `docs/public/`，已忽略。上游内置 LibSass 的弃用警告属于已知提示，应保留供审阅；不要添加 `--panicOnWarning`。新增内部链接使用 `relref`，避免写死根路径。

## Pages 工作流

`.github/workflows/docs-pages.yaml` 在 PR 中只构建；只有 `main` 的相关文件更新或从 `main` 手动运行才上传及部署。部署 job 单独声明 Pages 与 OIDC 权限，不添加自定义凭据。没有 `configure-pages` 自动启用步骤。

后续发布前需审阅内容、手动合并，并由有权限的维护者在仓库 Settings → Pages 选择 **GitHub Actions**。本次不修改该设置、不合并、不扩大权限。设置完成且工作流成功后，还需验证首页、目录、中文、搜索和 CSS/JS 资源。自定义域名或仓库改名时需另行更新 `hugo.yaml` 的 `baseURL`。

## 公开内容维护

每个业务设计页面保留“设计讨论稿 / 尚未实现”标识，直到结论与实现状态有证据支持。公开材料不包含私人聊天、内部标识、秘密及未授权资料。需求来源不足时写“待确认”，不要根据仓库名补造业务需求。

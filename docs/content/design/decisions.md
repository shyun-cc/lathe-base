---
title: 决策记录
weight: 70
---

**设计讨论稿 · 尚未实现（业务方案）。**

## ADR-001：文档先行

状态：本轮工作范围；业务设计仍待确认。

先提交设计讨论稿，再根据讨论结论提出业务实现。文档使用 `docs/` 下的独立站点，采用 OINK Hugo 主题，准备 GitHub Pages 工作流。PR 保持草稿，不自动合并。

## ADR-002：文档构建版本

状态：文档基础设施选型；不代表业务技术选型。

固定 OINK v1.5.1、Go 1.27.3、Hugo Extended 0.168.0，与核验时的官方 starter 一致。消费主题无需 Node.js、npm 或外部 Sass。代价是需要安装指定工具并下载模块；升级应通过独立 PR 验证。

依据：[OINK](https://github.com/pgsty/oink)、[OINK Starter](https://github.com/pgsty/oink-starter)。

## 后续 ADR 模板

编号、标题、状态（提议/接受/替代）、背景、候选方案、选择理由、影响、关联问题及验收条目。当前尚无已接受的业务决策。

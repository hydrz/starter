# 贡献指南

## 开始之前

1. 阅读[工程蓝图](docs/engineering-blueprint.md)与[工程规范索引](docs/standards/README.md)；
2. 从最新主分支创建范围单一的短生命周期分支；
3. 涉及不可逆或跨团队决策时，先提交 ADR；
4. API、数据库和生成代码变更分别遵循对应开发指南。

## 标准工作流

```bash
pnpm install --frozen-lockfile
pnpm format
pnpm check
pnpm test
pnpm build
```

数据库相关变更还必须执行 `pnpm test:database`。合并前使用[代码评审清单](docs/standards/code-review.md)完成作者自检。

## 变更约束

- Pull Request 只解决一个明确问题，不混入无关重构；
- 生成文件必须通过源文件或生成配置修改，禁止手工修补；
- 新行为必须有对应测试，修复缺陷时优先先添加回归测试；
- 不得提交密钥、生产数据、个人信息或本地 `.env`；
- 公共契约、migration 和运维行为发生变化时必须同步文档。

## 规范例外

无法遵守规范时，不应静默关闭检查。Pull Request 必须说明：

1. 违反的具体规则；
2. 当前无法遵守的原因与风险；
3. 替代控制措施；
4. 负责人和清理期限。

长期例外必须通过 ADR 接受；临时例外到期后应由跟踪任务移除。

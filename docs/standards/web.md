# Web 工程规范

## TypeScript

- 保持 `strict` 模式，不使用 `any` 绕过模型设计；不可信输入先以 `unknown` 接收并校验；
- API 类型和请求函数由 Orval 生成，禁止维护手写镜像 DTO；
- 使用判别联合、窄化和穷尽检查表达状态，避免无约束字符串；
- 模块只导出外部确实需要的成员，类型导入使用 `import type`；
- 禁止在业务代码中使用 `console.log`，可观测性通过约定的日志或错误边界接入。

## React

- 组件按业务 feature 组织；页面负责组合，数据与交互尽量留在所属 feature；
- 服务端状态由 TanStack Query 管理，组件不得把同一份远端数据复制到 Zustand；
- Zustand 仅用于跨页面、非服务端来源且确有共享需求的客户端状态；
- 表单使用 React Hook Form，边界校验使用 Zod；服务端仍必须重复验证安全与业务不变量；
- effect 仅用于和外部系统同步，不用于派生可在 render 中计算的数据；
- 异步界面必须提供加载、空、失败和成功反馈。

## 样式与可访问性

- 优先使用设计令牌与 shadcn/ui 组件，避免在业务组件中散落魔法颜色和尺寸；
- 表单控件必须有可访问名称，错误信息应与控件关联；
- 交互元素使用语义化 `button`、`a`、`input`，不得用 `div` 模拟；
- 键盘焦点必须可见，不能只用颜色传达状态；
- 页面需覆盖窄屏布局，重大视觉变更附截图验证。

## 最低验证

Web 变更至少运行 `pnpm check:web`、`pnpm --filter @enterprise/web test` 和 `pnpm build:web`。

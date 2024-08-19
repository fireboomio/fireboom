新增功能：
1. 订阅接口前置后置钩子支持
2. Rest数据源支持上传接口解析成graphql
3. Rest数据源支持additionalProperties解析成Map
4. 添加指令@whereInput用于注入参数构造查询条件
5. Proxy钩子支持上传/下载功能
6. Graphql跨源关联添加_join_mutation用于跨源变更操作
7. 添加支持多数据源事务管理(仅数据库类型)
8. 实现opentracing链路追踪并接入Prisma实现接口和SQL可观测的链路
9. 指令@Skip和@include添加参数ifRule使用表达式跳过或包含selection
10. 指令@export可以导出任何非空字段为true值参数
11. 添加指令@injectRuleValue用于注入参数可替换前置钩子功能
12. 新增接口并发控制，允许通过设置tickets和timeout来控制接口的同时请求数量
13. 添加queryRaw配合指令@customizedField实现返回字段自动补全功能，支持数组字段
14. 调整包含依赖@export导出值的参数的selection执行顺序，跨源关联现在将自动保证执行顺序
15. 实现Prisma引擎查询/内省部分错误的汉化功能，支持嵌套错误
16. 添加optional_queryRaw实现SQL片段可选的查询`&{}`直接替换, `${}`参数占位符
17. JSON类型入参可以直接传递JSON对象，无需再使用JSON.stringify()
18. 新增指令@skipVariable用于自定义规则跳过参数，例如当A参数为空时跳过B参数
19. 支持不带前端资源的精简部署，下载命令行添加--without-web参数，镜像末尾添加_without-web
20. 指令@injectCurrentDateTime新增toUnix用于注入当前时间戳，offset新增format用于偏移后重新格式化时间
21. 新增指令@firstRawResult用于获取RawQuery查询结果中的第一个元素，实现totalCount功能

破坏变更：
1. 修改JSON返回值不再是JSON字符串，而是JSON对象，方便前端使用，同时也支持@export指令导出JSON字段

Golang-sdk:
1. 添加支持手动管理事务函数
2. 添加调用上传接口，支持多文件上传
3. 优化自定义数据源内省结果比较，解决重复更新内省文件问题
4. 新增生成time.time和geometry(Postgres)字段类型
5. 添加支持根据struct构建Graphql的Args和Type
6. 解决MutatingPreResolve钩子丢失零值入参问题
7. 解决time.Time零值入参未被清除的问题
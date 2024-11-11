# 既往回顾

## 新增功能：

1. 订阅接口前置后置钩子支持
2. Rest数据源支持上传接口解析成graphql
3. Rest数据源支持additionalProperties解析成Map
4. 添加指令@whereInput用于注入参数构造查询条件
5. Proxy钩子支持上传/下载功能
6. Graphql跨源关联添加_join_mutation用于跨源变更操作
7. 添加支持多数据源事务管理(仅数据库类型)
8. 实现opentracing链路追踪并接入Prisma实现接口和SQL可观测的链路(jaegertracing/all-in-one:latest)
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

## 破坏变更：

1. 修改JSON返回值不再是JSON字符串，而是JSON对象，方便前端使用，同时也支持@export指令导出JSON字段

## Golang-sdk:

1. 添加支持手动管理事务函数
2. 添加调用上传接口，支持多文件上传
3. 优化自定义数据源内省结果比较，解决重复更新内省文件问题
4. 新增生成time.time和geometry(Postgres)字段类型
5. 添加支持根据struct构建Graphql的Args和Type
6. 解决MutatingPreResolve钩子丢失零值入参问题
7. 解决time.Time零值入参未被清除的问题

# 版本v2.2.0
## 新增功能:
1. 新增 Graphql 内置解析指令 @transform，替换原有根据 jsonpath 从根节点替换的功能
2. 全局 operation 和单个 operation 通过 graphqlTransformEnabled 切换新旧解析逻辑
3. Web 页面 Mock 执行 Graphql 新增按钮用于查看 @transform 指令解析后的数据
4. 端口 9123 新增接口 /api/storageClient/{dataName}/presignUpload 返回带签名认证的上传地址，用于前端直接上传文件
5. 新增指令 @asyncResolve 用于并行解析数组 selectionField，当数据量大时效果明显

## 升级功能:
1. 指令 @jsonschema 支持 format 和 enum 用于修改参数的类型，和指令 @hookVariable 结合更佳
2. 指令 @transform 新增参数 math(SUM/AVG/MAX/MIN/COUNT/FIRST/LAST)用于对数组元素进行计算，SUM/AVG/MAX/MIN 仅支持数字，FIRST 常用于取首个元素，针对一对多查询但只需取首个元素时使用

## 破坏变更:
1. query_raw 和 optional_query_raw 查询默认均解析成数组，可以通过 @firstRawResult 改变内省且取首个元素（计算 total 时常用）

## 问题修复:
1. 解决 Prisma Error P2024 错误未正确翻译的问题
2. 解决读取超长文件行错误的问题（当 schema definitions 非常大时出现）
3. 解决 Prisma 数据源为空时未能生成内省文件的问题（因为翻译错误内容但原有判断逻辑未修改导致）
4. Web 页面 Graphql Mock 预览数据时，修复以下被错误清除的 Variable/Selection, 指令 @hookVariable 修饰参数但参数有被 Graphql 使用，指令 @customizedField 修饰 Selection 但字段存在定义中
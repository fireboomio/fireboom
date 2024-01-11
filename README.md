
# Fireboom 介绍

Fireboom 是一个可视化的WEB API开发平台，前后端开发者都能使用。

![Fireboom 核心操作](https://www.fireboom.cloud/_next/image?url=%2F_next%2Fstatic%2Fmedia%2Fvisualization.2b31570f.gif&w=750&q=75)

**查看 [快速上手视频教程](https://www.bilibili.com/video/BV1rM411u7e8/?spm_id_from=888.80997.embed_other.whitelist&t=136)**


[![Open in Gitpod](https://gitpod.io/button/open-in-gitpod.svg)](https://gitpod.io/#https://github.com/fireboomio/fb-init-simple)

> [gitpod 介绍](https://juejin.cn/post/6844903773878386701)：Gitpod是一个在线IDE，可以从任何GitHub页面启动。在几秒钟之内，Gitpod就可以为您提供一个完整的开发环境，包括一个VS Code驱动的IDE和一个可以由项目定制化配置的云Linux容器。

**启动成功后，在 gitpod 底部切换到`PORTS`面板，选择 `9123` 端口打开即可**


前往->  [Fireboom 官网](https://www.fireboom.io/)

## 👀 Fireboom 是什么?

- Fb 是可视化 API 开发平台，可以快速开发 API；
- Fb 是一个可视化的 BaaS 开发平台（Backend as a Service)；
- Fb 是一个集「API 开发」、「身份验证」、「对象存储」等于一身的一站式开发平台；
- Fb 可以是中国版的 Hasura 和 Supabase，支持 PostgreSQL、MySQL、MongoDB 等数据库。

> 产品愿景：极致开发体验，`飞速布署`应用！

如果你喜欢如下产品，那你大概率也会喜欢 Fireboom 。

- [Hasura](https://hasura.io/)
- [Supabase](https://supabase.com/)
- [Firebase](https://firebase.google.cn)

## 🎉 Fireboom 有什么?

- 多数据源：数据库（PgSQL、MySQL、MongoDB）、REST API、GraphQL 以及消息队列等；
- 数据管理：简化版 Navicat，主要包含数据库建模和数据预览功能；
- 可视化构建 API：基于 GraphQL 可视化构建 API，支持 API 授权、跨源关联、数据缓存、N+1 查询等高阶能力；
- 实时推送：将 GET 请求转换为实时查询接口，同时具备实时推送能力，业务无死角；
- SDK 生成：根据 API 实时生成客户端 SDK，当前已支持 React SDK，计划支持 Vue SDK 和 Flutter SDK；
- 文件存储：集成 S3 规范，实现文件管理，后续将支持钩子进行文件上传的后置处理；
- 钩子机制：提供了灵活的钩子机制，具备 PRO CODE  能力 (Go、Node、Java、Python...)，无惧复杂业务。
- ...

## 👨谁适合使用 Fireboom ?

**前端开发者 + Fireboom（Node.js） = 全栈开发者**

- 可视化开发：可视化构建 API，前端也能驾驭
- PRO CODE：会写 Node TS 函数，就能定制业务逻辑
- SDK 生成：实时生成客户端 SDK，接口对接从未如此丝滑

**后端开发者 + Fireboom（Golang/Java/Python）= ∞**

- 声明式开发：声明式语言开发 API，BUG 更少
- 多语言支持：用任意后端语言编写钩子，Golang、Java、Python...
- 文档生成：实时生成 Swagger 文档，无需手工编写



**独立开发者 + Fireboom= 一支团队**

- 分钟级交付：将传统模式下 2 天才能完成的接口开发时间缩短至 2 分钟
- 一键部署：一键发布应用到 Sealos 平台，自动化运维无惧“三高”


**Hasura、Supabase 用户，获得更强大、快速的开发体验**

- Fb 更适用于本土开发者，中文学习资料及配套组件
- Fb 支持多种数据库，包括国内常用的 MySQL 数据库
- Fb 不引入额外学习成本，对外暴露 REST 端点，前端更友好
- Fb 权限系统更灵活，不仅支持数据库还支持 REST 和 GraphQL 数据源


## 💥 Fireboom 能用来做什么？

> Fireboom 是 BaaS 平台，理论上可以开发任意应用的 API！

**移动和 WEB 应用程序：**

Fireboom 擅长 API 构建，尤其擅长聚合不同库表或三方 API 的数据在一个请求中，能够节省网络请求的成本，提高应用性能。而大部分移动或 WEB 应用程序都是从数据库查询数据，这是 Fireboom 的强项。例如：[英语口语练习 APP](https://enjoyfreetalk.com/)

**中后台应用：**

Fireboom 能够与前端低代码平台结合，实现复杂业务逻辑。为了解决中后台开发的需求，Fireboom 生态集成了一套中后台管理界面，并与 Fireboom 深度打通。基于此，快速完成中后台应用，覆盖前端低代码无法实现的用例！例如：[Fireboom Admin](https://github.com/fireboomio/amis-admin)

**数据大屏应用：**

Fireboom 擅长数据聚合和复杂 SQL 查询，能够在一次查询中获得页面所需的全部数据，同时，Fireboom 支持服务端订阅，无需客户端轮询，即可实现大屏数据的实时更新。

**BFF 层：**

Fireboom 本身也是一个可编程网关，可作为各数据源的中央访问点，聚合不同数据，为不同客户端按需提供数据，同时提供鉴权等功能。

**物联网应用：**

Fireboom 支持消息队列，非常适合处理来自物联网设备的数据。Fireboom 将实时消息映射为 GraphQL 订阅，并以 REST API 的推送方式暴露给客户端。同时，Fireboom 支持开发者自定义脚本处理订阅事件，实现事件数据落库等功能。

## ❓ 为什么用 Fireboom？

首先，业务型 Web 应用 80% 由样板代码组成，例如增删改查，权限管理，用户管理，消息或者通知。一次又一次的建立这些功能，不仅乏味，而且减少了我们集中在软件与竞争对手不同之处的时间。

- 增删改查：绝大多数偏业务型项目，都是增删改查，复杂点的包括关联查询等
- 验证鉴权：所有生产型项目都需要身份验证和身份鉴权，且实现该功能需要耗费大量人力
- 文件存储：绝大数应用都需要文件存储，用来存储用户头像等，实现文件上传和管理也较为繁琐

其次，除了重复性工作，后端开发者往往还要实现非功能需求，这些需求不仅消耗大量精力，而且有一定的技术门槛。

- N+1 缓存：避免关联查询时重复查询数据的问题，提高应用性能
- 实时推送：对于 IM 聊天等应用，需要实现实时推送功能（传统方式需要使用 websocket 等技术）

最后，当前市场上存在诸多 API 开发框架，但这些框架大都基于某种特定编程语言实现，开发者掌握特定编程语言才能上手使用。

使用 Fireboom，
- 对于简单需求，无需掌握任何开发语言，只需了解数据库知识和 GraphQL 协议就能胜任
- 对于复杂需求，可编写钩子扩展逻辑，钩子基于 [HTTP 协议](https://docs.fireboom.io/jin-jie-gou-zi-ji-zhi/operation-gou-zi)，原则上兼容任意后端语言，此外 我们还实现了 Golang、Nodejs 的钩子 SDK 


## Fireboom 的核心架构？

**API 作为数据源和客户端的桥梁，目的是提供数据，而数据源往往有严苛的 schema ，API 本质上是 schema 的子集。** Fireboom 将数据源的 schema 以可视化的方式呈现，开发者通过界面勾选所需函数，构建客户端需要的 API 。

![Fireboom 架构图](https://www.fireboom.cloud/_next/static/media/framework.5ff914cd.svg)

Fireboom 采用声明式开发方式，它以 API 为中心，将所有数据源抽象为 API，包括 REST API、GraphQL API、数据库甚至消息队列等。通过统一协议 GraphQL 把他们聚合为“超图”，同时通过可视化界面，从“超图”中选择子集 Operation 作为函数签名，并将其编译为 REST-API。

开发者通过界面配置，即可开启某 API 的缓存或实时推送功能。

此外，Fireboom 基于 HTTP 协议实现了 HOOKS 机制，方便开发者采用任何喜欢的语言实现自定义逻辑。

# 快速上手

## Fireboom 服务
**安装 Fireboom**

```shell
curl -fsSL fireboom.io/install | bash -s project-name -t init-todo --cn
```

> 推荐使用 Github Codespace 快速体验下述流程！

**启动 Fireboom 服务**

```shell
./fireboom dev
```

启动成功日志：

```sh
Web server started on http://localhost:9123
```

**打开控制面板**

[http://localhost:9123](http://localhost:9123)

**更新Fireboom**

```shell
# 更新本地二进制命令
curl -fsSL https://www.fireboom.io/update | bash
```

## 钩子服务

Fireboom 的GraphQL OPERATION 可以构建绝大多数增删改查的需求（包括关联表查询或更新）。但若遇到 OPERATION 无法胜任的场景时，可使用钩子机制扩展逻辑。

![](https://2723694181-files.gitbook.io/~/files/v0/b/gitbook-x-prod.appspot.com/o/spaces%2FNx22Cp3wzkuW1siRbMwW%2Fuploads%2Fgit-blob-24c89a58be58a1feadda5631d0781b74ef2b6dc7%2Fimage%20(2)%20(1)%20(1)%20(1)%20(1)%20(1)%20(1)%20(1).png?alt=media)

目前已支持NodeJS、Golang、Java 语言的SDK，其他未提供SDK 的语言，可基于HTTP规范自行开发。

### 安装钩子

![安装钩子](https://2723694181-files.gitbook.io/~/files/v0/b/gitbook-x-prod.appspot.com/o/spaces%2FNx22Cp3wzkuW1siRbMwW%2Fuploads%2Fgit-blob-1faf4f6d4e7d0a8bf07133971e02a019188f0c1e%2Fimage%20(55).png?alt=media)

1. 点击<状态栏>的<钩子模版:未选择>，进入模板页
2. 点击右上角<浏览模板市场>，打开模板市场
3. 在**钩子模板**分组下载对应SDK（根据你的语言选择），目录 template 下新建对应文件夹

ps：**不建议钩子开发过程中切换钩子的语言！** 否则，已开启钩子需要用新语言重新编写。

### Golang 钩子

1. 开启 `Golang server` 钩子

根目录下新建`custom-go`文件夹 

2.安装 golang 依赖
```sh
# 进入 custom-go 目录
cd custom-go/
# 安装依赖
go mod tidy
```
3. 编写局部钩子

在[API管理]TAB，选择 `Todo/CreateOneTodo` 接口，打开 `postResolve` 钩子。

可以看到 `custom-go/operation/Todo/CreateOneTodo/postResolve.go` 文件。

将其修改为：

```go
package CreateOneTodo

import (
	"custom-go/generated"
	"custom-go/pkg/base"
	"fmt"
)

func PostResolve(hook *base.HookRequest, body generated.Todo__CreateOneTodoBody) (res generated.Todo__CreateOneTodoBody, err error) {
	// body 挂载了对象，如 入参 input、响应 resopnse
	fmt.Println("Input", body.Input)
	fmt.Println("Response", body.Response)
	// hook 挂载了其他对象，如 登录用户 user
	fmt.Println("User", hook.User)
	// if err != nil {
	// 	hook.Logger().Errorf(err.Error())
	// }
	return body, nil
}
```
4. 编写funtion钩子

在[数据源]TAB，点击 <脚本->Function> 新建 Function 钩子，命名为 hello。

可以看到 custom-go/function/hello.go 文件。

这是一个用户登录的逻辑 ~
```go
package function
import (
	"custom-go/pkg/base"
	"custom-go/pkg/plugins"
	"custom-go/pkg/wgpb"
)

func init() {
	plugins.RegisterFunction[hello_loginReq, hello_loginRes](hello, wgpb.OperationType_MUTATION)
}

type hello_loginReq struct {
	Username string    `json:"username"`
	Password string    `json:"password"`
	Info     hello_loginInfo `json:"info,omitempty"`
}

type hello_loginInfo struct {
	Code    string `json:"code,omitempty"`
	Captcha string `json:"captcha,omitempty"`
}

type hello_loginRes struct {
	Msg  string `json:"msg"`
	Data string `json:"data"`
}

func hello(hook *base.HookRequest, body *base.OperationBody[hello_loginReq, hello_loginRes]) (*base.OperationBody[hello_loginReq, hello_loginRes], error) {
	if body.Input.Username != "John" || body.Input.Password != "123456" {
		body.Response = &base.OperationBodyResponse[hello_loginRes]{
			Errors: []base.GraphQLError{{Message: "username or password wrong"}},
		}
		return body, nil
	}

	body.Response = &base.OperationBodyResponse[hello_loginRes]{Data: hello_loginRes{Msg: "login success"}}
	return body, nil
}
```

**打开 custom-go/main.go 文件，打开第6行注释**，引入上述包

```go
package main

import (
	// 根据需求，开启注释
	//_ "custom-go/customize"
	_ "custom-go/function" // 开启后function 才生效
	//_ "custom-go/proxy"
	"custom-go/server"
)

func main() {
	server.Execute()
}
```


5. 启动钩子

```sh
go run main.go
```

6. 测试钩子
7. 
**局部钩子**

执行请求：
```sh
curl 'http://127.0.0.1:9991/operations/Todo/CreateOneTodo' \
  -X POST  \
  -H 'Content-Type: application/json' \
  --data-raw '{"title":"learn fireboom"}' \
  --compressed
```

输出响应：

```json
{"data":{"data":{"id":9,"title":"learn fireboom","completed":false,"createdAt":"2024-01-11T16:04:55.286Z"}}}
```
钩子控制台：
```log
Input {learn fireboom}
Response &{<nil> {{false 2024-01-11T16:04:55.286Z 9 learn fireboom}} []}
User <nil>
```

**Function 钩子**

执行请求：
```sh
curl http://127.0.0.1:9991/operations/function/hello \
  -X POST \
  -H 'Content-Type: application/json' \
  --data-raw '{"info":{"captcha":"string","code":"string"},"password":"string","username":"string"}' \
  --compressed
```

响应结果：
```log
{"data":{"data":"","msg":""},"errors":[{"message":"username or password wrong","path":null}]}
```

### NodeJS 钩子

1. 开启 `node-server` 钩子

根目录下新建`custom-ts`文件夹 

2. 安装 nodejs 依赖
```sh
# 进入 custom-ts 目录
cd custom-ts/
# 安装依赖
npm i
```
3. 编写局部钩子

在[API管理]TAB，选择 `Todo/CreateOneTodo` 接口，打开 `postResolve` 钩子。

可以看到 `custom-ts/operation/Todo/CreateOneTodo/postResolve.ts` 文件。

将其修改为：

```ts
import { registerPostResolve } from '@fireboom/server'
import { type FireboomOperationsDefinition } from '@/operations'
import { Todo__CreateOneTodoInput, Todo__CreateOneTodoResponseData } from '@/models'

registerPostResolve<Todo__CreateOneTodoInput, Todo__CreateOneTodoResponseData, FireboomOperationsDefinition>('Todo/CreateOneTodo', async ctx => {
	// ctx 挂载了对象，如 入参 input、响应 resopnse、登录用户 user
    console.log("input:",ctx.input)
    console.log("response:",ctx.response)
    console.log("user:",ctx.user)
  return ctx.response
})
```

4. 编写funtion钩子

在[数据源]TAB，点击 <脚本->Function> 新建 Function 钩子，命名为 hello。

可以看到 custom-ts/function/hello.ts 文件。

这是一个推流函数 ~
```ts
import { OperationType, registerFunctionHandler } from '@fireboom/server'
import { type FireboomRequestContext } from '@/operations'
registerFunctionHandler('hello', {
  input: {
    type: 'object',
    properties: {
      "name": {
        type: 'string'
      }
    },
    additionalProperties: false
  },
  response: {
    // only support object as root
    type: 'object',
    properties: {
      "msg": {
        type: 'string'
      }
    }
  },
  operationType: OperationType.SUBSCRIPTION, // 订阅类型
  handler: async function* (input, ctx: FireboomRequestContext) {
    for (let i = 0; i < 10; i++) {
      yield { msg: `Hello ${i}` }
      await new Promise((resolve) => setTimeout(resolve, 1000))
    }
  }
})
```

5. 启动钩子

```sh
npm run dev
```

6. 测试钩子

**局部钩子**

执行请求：
```sh
curl 'http://127.0.0.1:9991/operations/Todo/CreateOneTodo' \
  -X POST  \
  -H 'Content-Type: application/json' \
  --data-raw '{"title":"learn fireboom"}' \
  --compressed
```

输出响应：

```json
{"data":{"data":{"id":5,"title":"learn fireboom","completed":false,"createdAt":"2024-01-10T16:17:08.883Z"}}}
```
钩子控制台：
```log
input: { title: 'learn fireboom' }
response: {
  data: {
    data: {
      id: 8,
      title: 'learn fireboom',
      completed: false,
      createdAt: '2024-01-10T16:22:53.272Z'
    }
  }
}
user: undefined
```

**Function 钩子**

在网页访问：

```http
GET http://127.0.0.1:9991/operations/function/hello?wg_variables={%22name%22:%22string%22}&wg_sse=true
```

结果：
```log
data: {"data":{"msg":"Hello 0"}}
data: {"data":{"msg":"Hello 1"}}
data: {"data":{"msg":"Hello 2"}}
...
data: {"data":{"msg":"Hello 8"}}
data: {"data":{"msg":"Hello 9"}}
data: done
```

# 参考

- [Fireboom 官网](https://www.fireboom.cloud)
- [Fireboom 文档中心](https://docs.fireboom.io/)
- [Fireboom 视频教程](https://space.bilibili.com/3493080529373820/channel/collectiondetail?sid=1505636)

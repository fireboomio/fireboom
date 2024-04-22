# Fireboom - 开发者入门指南

欢迎来到Fireboom项目！这份入门指南旨在帮助您快速初始化项目并在本地开发环境中运行。

### 先决条件

在开始之前，请确认您的系统已安装了以下软件：

- Git
- Go 1.21 或更高版本

### 克隆仓库

首先，请使用以下命令克隆Fireboom仓库：

```shell
git clone --recurse-submodules https://github.com/fireboomio/fireboom.git
```

这里的`--recurse-submodules`选项非常重要，因为它同时会克隆项目中包含的 `fireboom-web` 和 `wundergraph` 两个子模块。

### 子模块

Fireboom项目包含以下两个子模块：

1. **前端资源目录（`assets/front`）：**

   此子模块包含项目所需的所有前端资源。
   Git仓库地址：https://github.com/fireboomio/fireboom-web.git

2. **后端资源目录（`wundergraphGitSubmodule`）：**

   此子模块包含从其他开源项目中fork的后端资源。
   Git仓库地址：https://github.com/fireboomio/wundergraph.git

为确保子模块是最新的，请执行以下命令：

```shell
git submodule update --init --recursive
```

### 编译项目

Fireboom项目使用Cobra构建命令行参数解析。第一个参数是必填的，可以是 `dev`、`start` 或 `build`。您还可以选择额外的参数来覆盖环境变量或切换其他功能。

请在项目根目录下运行以下命令来编译项目：

```shell
go build -o fireboom
```

### 运行项目

编译完成后，您可以运行项目，使用以下命令之一：

```shell
./fireboom dev
```

或

```shell
./fireboom start
```

或

```shell
./fireboom build
```

您可以根据需要添加其他参数来自定义环境设置或功能开关。

### 寻求帮助

如果在使用命令时遇到任何问题或需求帮助，请查看Cobra文档或在GitHub仓库上提出issue。

### 贡献代码

欢迎为Fireboom项目贡献代码！请先详细阅读我们的贡献指南，再开始进行Pull Request。

### 许可协议

该项目是开源的，并遵循[LICENSE](LICENSE)许可协议。在贡献或使用本项目时，请确保您遵守了许可协议。

### 联系方式

如果您有任何问题或需要协助，请在GitHub仓库中提出issue，我们的团队将会与您联系。

期待您在Fireboom项目中的工作！您的贡献是我们及开源社区的宝贵财富。
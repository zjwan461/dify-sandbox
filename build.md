# Dify Sandbox 构建指南

本文档详细说明 Dify Sandbox 项目的镜像构建体系，涵盖构建流程、文件结构、版本管理及多种构建方式。

---

## 一、构建体系概览

项目采用**模板化 + 版本配置集中管理**的 Docker 镜像构建体系，提供两种构建方式：

| 方式 | 适用场景 | 宿主机依赖 |
|------|----------|------------|
| **All-in-One 构建**（推荐） | 本地手动构建 | 仅需 Docker |
| **传统分步构建** | CI/CD 或需要精细控制 | 需安装 Go、yq、gcc 等 |

---

## 二、核心文件结构

```
docker/
├── versions.yaml                          # 版本配置文件（集中管理所有软件版本）
├── generate.sh                            # Dockerfile 生成脚本（传统方式用）
├── entrypoint.sh                          # 容器入口脚本
├── production-all-in-one.dockerfile       # All-in-One 多阶段构建 Dockerfile
└── templates/
    ├── base.dockerfile                    # 基础镜像模板
    ├── production.dockerfile              # 生产环境模板（传统方式用）
    └── test.dockerfile                    # 测试环境模板（多阶段构建+测试）

build/
├── build_all_in_one.sh                    # All-in-One 构建脚本（推荐）
├── build_amd64.sh                         # AMD64 架构编译脚本
├── build_arm64.sh                         # ARM64 架构编译脚本
└── build_arm64_permission_test.sh         # ARM64 权限测试编译脚本

.github/workflows/
├── build.yml                              # 单架构构建 workflow
├── build-universal.yml                    # 多架构通用构建 workflow
└── tests.yml                              # 测试 workflow
```

---

## 三、版本配置

所有软件版本在 [`docker/versions.yaml`](docker/versions.yaml) 中集中管理：

```yaml
versions:
  python: "dhi.io/python:3-debian13-sfw-ent-dev"   # Python 基础镜像
  golang: "1.25.0"                                   # Go 编译版本
  nodejs: "v20.20.0"                                 # Node.js 版本
  python_packages: "httpx==0.27.2 requests==2.33.0 jinja2==3.1.6 PySocks httpx[socks]"

mirrors:
  debian: "http://deb.debian.org/debian testing main"
  nodejs: "https://npmmirror.com/mirrors/node"
  golang: "https://golang.org/dl"
```

> **修改版本号时**，只需编辑此文件，无需改动 Dockerfile 或构建脚本。

---

## 四、All-in-One 构建方式（推荐）

### 特点
- **零宿主机依赖**：无需安装 Go、gcc、yq 等工具，只需 Docker
- **一条命令**：自动完成编译和镜像构建
- **多阶段构建**：编译和运行环境完全隔离

### 快速开始

```bash
# 仅本地构建
./build/build_all_in_one.sh -n dify-sandbox -t v1.0.0

# 构建并推送到远端
./build/build_all_in_one.sh -n my-registry/dify-sandbox -t v1.0.0 --push
```

### 脚本参数

| 参数 | 说明 | 是否必填 |
|------|------|----------|
| `-n, --name` | 镜像名称 | 必填 |
| `-t, --tag` | 镜像标签 | 必填 |
| `-f, --file` | Dockerfile 路径（默认 `docker/production-all-in-one.dockerfile`） | 可选 |
| `--push` | 构建后推送到远端仓库 | 可选 |
| `-h, --help` | 显示帮助信息 | 可选 |

### 构建流程

```
Stage 1 (builder: golang:1.25.0)       Stage 2 (production: python:3-debian13)
┌──────────────────────────┐            ┌──────────────────────────┐
│  1. 安装 gcc/libseccomp  │            │  1. 安装系统依赖          │
│  2. 编译 python.so       │   COPY     │  2. 安装 Python 依赖包    │
│  3. 编译 nodejs.so       │ ────────→  │  3. 下载 Node.js          │
│  4. 编译 main            │  main/env  │  4. 运行 env 初始化       │
│  5. 编译 env             │            │  5. 设置入口脚本          │
└──────────────────────────┘            └──────────────────────────┘
```

### 更多示例

```bash
# 推送到 Docker Hub
./build/build_all_in_one.sh -n langgenius/dify-sandbox -t v0.1.0 --push

# 推送到私有仓库
./build/build_all_in_one.sh -n registry.example.com/dify-sandbox -t v1.0.0 --push

# 使用自定义 Dockerfile
./build/build_all_in_one.sh -n dify-sandbox -t latest -f docker/custom.dockerfile
```

---

## 五、传统分步构建方式

适用于 CI/CD 流程或需要精细控制构建过程的场景。

### 前置依赖

- **Go**（版本与 `versions.yaml` 一致）
- **yq**（YAML 解析工具）
- **gcc**、**libseccomp-dev**（CGO 编译依赖）

### 步骤

#### 1. 编译 Go 二进制文件

```bash
# AMD64
bash ./build/build_amd64.sh

# ARM64
bash ./build/build_arm64.sh
```

编译产物：

| 文件 | 说明 |
|------|------|
| `internal/core/runner/python/python.so` | Python 运行时 CGO 共享库 |
| `internal/core/runner/nodejs/nodejs.so` | Node.js 运行时 CGO 共享库 |
| `main` | 主服务二进制文件 |
| `env` | 环境初始化程序 |

#### 2. 生成 Dockerfile

```bash
cd docker
./generate.sh production amd64    # 或 arm64
```

脚本会读取 `versions.yaml` 中的版本号，将模板中的占位符替换为实际值，输出文件如 `amd64-production.gen.dockerfile`。

#### 3. 构建 Docker 镜像

```bash
docker build -t dify-sandbox -f ./docker/amd64-production.gen.dockerfile .
```

#### 4. 打 Tag 并推送（可选）

```bash
docker tag dify-sandbox your-registry/dify-sandbox:v1.0.0-amd64
docker push your-registry/dify-sandbox:v1.0.0-amd64
```

---

## 六、CI/CD 自动构建

### 触发条件

| 事件 | 触发的 Workflow |
|------|-----------------|
| Push 到 `main` 分支 | `build-universal.yml` → 构建 amd64 + arm64 + 合并通用镜像 |
| 发布 Release（打 tag） | `build-universal.yml` → 同上，额外打 `latest` 和版本 tag |
| Pull Request 到 `main` | `tests.yml` → 构建测试镜像并运行集成测试 |

### 镜像 Tag 生成规则

由 `docker/metadata-action@v5` 自动生成：

| 触发场景 | 生成的 Tag |
|---------|-----------|
| Push 到 main 分支 | `main` + commit SHA（如 `abc123...`） |
| 发布 Release | `latest` + tag 名称（如 `v0.1.0`）+ commit SHA |
| 每次构建 | 追加架构后缀 `-amd64` 或 `-arm64` |

### 镜像名称

```yaml
DOCKERHUB_IMAGE: ${{ vars.DIFY_SANDBOX_IMAGE_NAME || 'langgenius/dify-sandbox' }}
```

默认推送到 Docker Hub（`langgenius/dify-sandbox`）和 AWS ECR。

### 多架构合并流程

```
build-amd64 (depot-ubuntu-24.04-4)  ──→  推送 :xxx-amd64
build-arm64 (arm64_runner)          ──→  推送 :xxx-arm64
         │
         ▼
build-universal ──→  docker manifest create 合并 → 推送 :xxx (通用镜像)
```

---

## 七、测试构建

测试环境使用 [`docker/templates/test.dockerfile`](docker/templates/test.dockerfile)，采用多阶段构建在容器内完成编译和测试：

```bash
# 生成测试 Dockerfile
cd docker && ./generate.sh test amd64

# 构建并运行测试
docker build -t test -f docker/amd64-test.gen.dockerfile .
docker run --rm test
```

测试 Dockerfile 会在容器内：
1. 使用 `golang` 镜像编译所有 Go 产物
2. 切换到 Python 基础镜像
3. 安装系统依赖、Go、Node.js
4. 运行集成测试（`go test -timeout 120s`）

---

## 八、容器入口脚本

[`docker/entrypoint.sh`](docker/entrypoint.sh) 在容器启动时执行：

```bash
#!/bin/bash
# 1. 解压 Node.js
tar -xvf $NODE_TAR_XZ -C /opt
# 2. 创建 node 软链接
mkdir -p /usr/local/bin/
ln -s $NODE_DIR/bin/node /usr/local/bin/node
rm -f $NODE_TAR_XZ
# 3. 启动主服务
/main
```

> Node.js 在构建时下载压缩包，在容器首次启动时解压，以减少镜像层体积。

---

## 九、架构支持

| 架构 | Go 编译目标 | Node.js 包名 |
|------|------------|-------------|
| AMD64 | `linux/amd64` | `node-v20.20.0-linux-x64` |
| ARM64 | `linux/arm64` | `node-v20.20.0-linux-arm64` |

---

## 十、常见问题

### Q: 宿主机没有 Go 环境怎么构建？
使用 All-in-One 方式，只需 Docker 即可：
```bash
./build/build_all_in_one.sh -n dify-sandbox -t latest
```

### Q: 如何修改 Python/Node.js/Go 版本？
编辑 [`docker/versions.yaml`](docker/versions.yaml)，所有版本号集中在此管理。

### Q: 如何推送到私有镜像仓库？
```bash
./build/build_all_in_one.sh -n registry.example.com/dify-sandbox -t v1.0.0 --push
```

### Q: 如何在 amd64 机器上交叉编译 arm64 镜像？
传统方式需要安装交叉编译工具链。All-in-One 方式可通过 Docker 的 `--platform` 参数：
```bash
docker buildx build --platform linux/arm64 -f docker/production-all-in-one.dockerfile -t dify-sandbox:arm64 .
```

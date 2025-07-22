# Docker 容器管理模块

本模块为 CI/CD 服务提供完整的 Docker 容器管理功能，支持容器的完整生命周期管理、资源监控、自动清理等企业级特性。

## 📁 模块结构

```
docker/
├── docker_manager.go          # 核心接口定义和数据结构
├── docker_manager_impl.go     # 容器生命周期管理实现
├── docker_manager_extended.go # 扩展功能实现
└── README.md                  # 本文档
```

## 🔧 核心功能

### 1. 容器生命周期管理
- **创建容器**: 支持完整的容器配置，包括资源限制、网络设置、卷挂载等
- **启动/停止/重启**: 完整的容器控制操作
- **删除容器**: 支持强制删除和优雅删除

### 2. 镜像管理
- **拉取镜像**: 支持从 Docker Hub 和私有仓库拉取镜像
- **构建镜像**: 支持从 Dockerfile 构建自定义镜像
- **删除镜像**: 清理未使用的镜像
- **镜像列表**: 获取本地镜像清单

### 3. 网络管理
- **创建网络**: 支持多种网络驱动（bridge、overlay 等）
- **网络连接**: 容器与网络的动态连接和断开
- **网络隔离**: 支持安全的网络隔离配置

### 4. 资源监控
- **实时统计**: CPU、内存、网络、磁盘 I/O 统计
- **资源限制**: 支持 CPU 和内存限制
- **健康检查**: 容器和系统健康状态监控

### 5. 自动化管理
- **自动清理**: 定期清理已停止的容器、未使用的镜像和网络
- **资源池管理**: 内存中维护容器和网络池，提高查询效率
- **统计收集**: 后台收集容器运行统计信息

## 🚀 快速开始

### 基本用法

```go
import "github.com/cloud-platform/collaborative-dev/internal/cicd-service/docker"

// 创建 Docker 管理器
config := docker.DefaultManagerConfig()
logger := zap.NewDevelopment()

dockerManager, err := docker.NewDockerManager(config, logger)
if err != nil {
    log.Fatal(err)
}
defer dockerManager.Close()

// 创建容器配置
containerConfig := &docker.ContainerConfig{
    Name:    "my-build-container",
    Image:   "ubuntu",
    Tag:     "20.04",
    Cmd:     []string{"/bin/bash", "-c", "echo 'Hello CI/CD!'"},
    Env:     []string{"CI=true", "BUILD_ID=123"},
    
    // 资源限制
    CPULimit:    1.0,     // 1 CPU 核心
    MemoryLimit: 512 * 1024 * 1024, // 512MB
    
    // 卷挂载
    Volumes: map[string]string{
        "/host/workspace": "/workspace",
    },
    
    // 端口映射
    Ports: map[string]string{
        "8080": "8080",
    },
}

// 创建并启动容器
ctx := context.Background()
container, err := dockerManager.CreateContainer(ctx, containerConfig)
if err != nil {
    log.Fatal(err)
}

err = dockerManager.StartContainer(ctx, container.ID)
if err != nil {
    log.Fatal(err)
}

// 监控容器
stats, err := dockerManager.GetContainerStats(ctx, container.ID)
if err == nil {
    fmt.Printf("CPU: %.2f%%, Memory: %d bytes\n", 
        stats.CPUUsage, stats.MemoryUsage)
}

// 获取日志
logs, err := dockerManager.GetContainerLogs(ctx, container.ID, &docker.LogOptions{
    ShowStdout: true,
    ShowStderr: true,
    Follow:     false,
})
if err == nil {
    io.Copy(os.Stdout, logs)
    logs.Close()
}

// 清理
dockerManager.StopContainer(ctx, container.ID, 10*time.Second)
dockerManager.RemoveContainer(ctx, container.ID, false)
```

## ⚙️ 配置选项

### ManagerConfig 配置结构

```go
type ManagerConfig struct {
    DockerHost         string        // Docker 主机地址
    APIVersion         string        // Docker API 版本
    TLSCert           string        // TLS 证书路径
    TLSKey            string        // TLS 密钥路径
    TLSCACert         string        // TLS CA 证书路径
    MaxContainers     int           // 最大容器数量
    DefaultTimeout    time.Duration // 默认超时时间
    CleanupInterval   time.Duration // 清理间隔
    StatsInterval     time.Duration // 统计收集间隔
    EnableMonitoring  bool          // 启用监控
    EnableAutoCleanup bool          // 启用自动清理
}
```

### 默认配置

```go
config := docker.DefaultManagerConfig()
// 使用以下默认值：
// - DockerHost: "unix:///var/run/docker.sock"
// - APIVersion: "1.41"
// - MaxContainers: 100
// - DefaultTimeout: 30s
// - CleanupInterval: 5min
// - StatsInterval: 10s
// - EnableMonitoring: true
// - EnableAutoCleanup: true
```

## 🔍 监控和统计

### 容器统计信息

```go
stats, err := dockerManager.GetContainerStats(ctx, containerID)
if err == nil {
    fmt.Printf("容器统计:\n")
    fmt.Printf("  CPU 使用率: %.2f%%\n", stats.CPUUsage)
    fmt.Printf("  内存使用: %d / %d bytes\n", stats.MemoryUsage, stats.MemoryLimit)
    fmt.Printf("  网络接收: %d bytes\n", stats.NetworkRx)
    fmt.Printf("  网络发送: %d bytes\n", stats.NetworkTx)
    fmt.Printf("  磁盘读取: %d bytes\n", stats.DiskRead)
    fmt.Printf("  磁盘写入: %d bytes\n", stats.DiskWrite)
}
```

### 系统信息

```go
systemInfo, err := dockerManager.GetSystemInfo(ctx)
if err == nil {
    fmt.Printf("系统信息:\n")
    fmt.Printf("  运行容器: %d\n", systemInfo.ContainersRunning)
    fmt.Printf("  总镜像数: %d\n", systemInfo.Images)
    fmt.Printf("  总内存: %d bytes\n", systemInfo.MemTotal)
    fmt.Printf("  CPU 核心: %d\n", systemInfo.CPUs)
    fmt.Printf("  Docker 版本: %s\n", systemInfo.DockerVersion)
}
```

### 实时统计收集

模块内置统计收集器，自动收集运行容器的统计信息：

```go
// 获取统计通道
statsChannel := dockerManager.GetStatsCollector().GetStats()

// 处理统计信息
go func() {
    for stats := range statsChannel {
        // 处理统计数据
        processStats(stats)
    }
}()
```

## 🛠️ 高级功能

### 1. 容器过滤查询

```go
filter := &docker.ContainerFilter{
    Status: []string{"running", "paused"},
    Labels: map[string]string{
        "job_id": "job-123",
        "pipeline": "main",
    },
    Limit: 10,
}

containers, err := dockerManager.ListContainers(ctx, filter)
```

### 2. 镜像构建

```go
buildOptions := &docker.BuildOptions{
    Tags:       []string{"my-app:latest", "my-app:v1.0"},
    Dockerfile: "Dockerfile.prod",
    NoCache:    true,
    BuildArgs: map[string]string{
        "VERSION": "1.0.0",
        "ENV":     "production",
    },
}

buildOutput, err := dockerManager.BuildImage(ctx, buildContext, buildOptions)
```

### 3. 网络管理

```go
// 创建网络
networkOptions := &docker.NetworkOptions{
    Driver:     "bridge",
    Internal:   false,
    Attachable: true,
    Labels: map[string]string{
        "project": "cicd-platform",
    },
}

network, err := dockerManager.CreateNetwork(ctx, "cicd-network", networkOptions)

// 连接容器到网络
err = dockerManager.ConnectToNetwork(ctx, network.ID, container.ID)
```

### 4. 健康检查

```go
healthCheck := &docker.HealthCheckConfig{
    Test:        []string{"CMD-SHELL", "curl -f http://localhost:8080/health || exit 1"},
    Interval:    30 * time.Second,
    Timeout:     10 * time.Second,
    Retries:     3,
    StartPeriod: 60 * time.Second,
}

containerConfig.HealthCheck = healthCheck
```

## 🔒 安全特性

### 1. 安全配置
- **非特权运行**: 默认以非特权用户运行容器
- **只读根文件系统**: 支持只读根文件系统配置
- **安全选项**: 支持各种 Docker 安全选项
- **资源限制**: 严格的 CPU 和内存限制

### 2. 网络隔离
- **自定义网络**: 支持创建隔离的网络环境
- **端口控制**: 精确控制端口映射
- **防止特权升级**: 默认禁用新特权获取

### 3. 资源保护
- **容器数量限制**: 防止资源耗尽
- **自动清理**: 防止磁盘空间泄漏
- **超时控制**: 防止长时间运行的容器

## 📊 性能特性

### 1. 并发安全
- **线程安全**: 所有操作都是线程安全的
- **读写锁**: 使用读写锁优化并发访问
- **连接池**: 高效的 Docker 客户端管理

### 2. 资源优化
- **内存池**: 容器和网络对象池化管理
- **批量操作**: 支持批量容器操作
- **智能缓存**: 缓存容器状态信息

### 3. 监控优化
- **异步收集**: 统计信息异步收集，不阻塞主流程
- **采样率控制**: 可配置的统计采样间隔
- **资源使用优化**: 最小化监控对系统资源的影响

## 🐛 错误处理

模块提供完整的错误处理机制：

```go
// 所有方法都返回具体的错误信息
container, err := dockerManager.CreateContainer(ctx, config)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "容器数量已达上限"):
        // 处理容器数量限制
    case strings.Contains(err.Error(), "镜像不存在"):
        // 处理镜像问题
    default:
        // 处理其他错误
    }
}
```

## 📝 日志记录

模块使用结构化日志记录所有重要操作：

```go
// 日志级别和内容示例
INFO  - 容器创建成功 {"container_id": "abc123", "name": "build-job"}
WARN  - 系统内存使用率过高 {"usage_percent": 92.5}
ERROR - 清理容器失败 {"container_id": "def456", "error": "permission denied"}
```

## 🔧 故障排除

### 常见问题

1. **Docker 连接失败**
   - 检查 Docker 服务是否运行
   - 验证 Docker socket 权限
   - 检查网络连接

2. **容器创建失败**
   - 检查镜像是否存在
   - 验证资源限制设置
   - 检查端口冲突

3. **统计收集问题**
   - 检查容器是否运行
   - 验证 Docker API 权限
   - 检查系统资源

### 调试模式

启用详细日志记录：

```go
logger, _ := zap.NewDevelopment()
dockerManager, _ := docker.NewDockerManager(config, logger)
```

## 🚀 最佳实践

1. **资源管理**
   - 设置合理的资源限制
   - 定期清理不需要的容器和镜像
   - 监控系统资源使用情况

2. **安全配置**
   - 使用非特权用户运行容器
   - 限制容器网络访问
   - 定期更新基础镜像

3. **性能优化**
   - 使用适当的镜像大小
   - 优化 Dockerfile 构建
   - 合理设置清理间隔

4. **错误处理**
   - 实现完整的错误恢复机制
   - 记录详细的操作日志
   - 设置合理的超时时间

## 📈 扩展性

模块设计支持以下扩展：

1. **多 Docker 主机**: 可扩展支持多个 Docker 主机
2. **编排集成**: 可集成 Kubernetes 或 Docker Swarm
3. **插件系统**: 支持自定义插件扩展功能
4. **指标导出**: 可集成 Prometheus 等监控系统

这个模块为 CI/CD 服务提供了强大、安全、高性能的容器管理能力，是构建现代化持续集成平台的重要基础组件。
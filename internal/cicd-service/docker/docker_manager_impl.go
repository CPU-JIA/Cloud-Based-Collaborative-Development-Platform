package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/go-connections/nat"
	"go.uber.org/zap"
)

// CreateContainer 创建容器
func (dm *dockerManager) CreateContainer(ctx context.Context, config *ContainerConfig) (*Container, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	
	// 检查容器数量限制
	if len(dm.containerPool) >= dm.config.MaxContainers {
		return nil, fmt.Errorf("容器数量已达上限: %d", dm.config.MaxContainers)
	}
	
	// 构建容器配置
	containerConfig := &container.Config{
		Image:        fmt.Sprintf("%s:%s", config.Image, config.Tag),
		Cmd:          config.Cmd,
		Env:          config.Env,
		WorkingDir:   config.WorkingDir,
		Labels:       config.Labels,
		ExposedPorts: make(nat.PortSet),
		User:         config.User,
	}
	
	// 设置端口映射
	portBindings := make(nat.PortMap)
	for containerPort, hostPort := range config.Ports {
		port, err := nat.NewPort("tcp", containerPort)
		if err != nil {
			return nil, fmt.Errorf("无效的端口配置: %v", err)
		}
		containerConfig.ExposedPorts[port] = struct{}{}
		portBindings[port] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: hostPort,
			},
		}
	}
	
	// 设置健康检查
	if config.HealthCheck != nil {
		containerConfig.Healthcheck = &container.HealthConfig{
			Test:        config.HealthCheck.Test,
			Interval:    config.HealthCheck.Interval,
			Timeout:     config.HealthCheck.Timeout,
			Retries:     config.HealthCheck.Retries,
			StartPeriod: config.HealthCheck.StartPeriod,
		}
	}
	
	// 构建主机配置
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Binds:        []string{},
		NetworkMode:  container.NetworkMode(config.NetworkMode),
		AutoRemove:   config.AutoRemove,
		Privileged:   config.Privileged,
		ReadonlyRootfs: config.ReadOnly,
		SecurityOpt:  config.SecurityOpts,
	}
	
	// 设置卷挂载
	for hostPath, containerPath := range config.Volumes {
		hostConfig.Binds = append(hostConfig.Binds, fmt.Sprintf("%s:%s", hostPath, containerPath))
	}
	
	// 设置资源限制
	if config.CPULimit > 0 {
		hostConfig.Resources.NanoCPUs = int64(config.CPULimit * 1e9)
	}
	if config.MemoryLimit > 0 {
		hostConfig.Resources.Memory = config.MemoryLimit
	}
	
	// 设置重启策略
	if config.RestartPolicy != "" {
		hostConfig.RestartPolicy = container.RestartPolicy{
			Name: config.RestartPolicy,
		}
	}
	
	// 创建容器
	resp, err := dm.client.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		&network.NetworkingConfig{},
		nil,
		config.Name,
	)
	if err != nil {
		return nil, fmt.Errorf("创建容器失败: %v", err)
	}
	
	// 构建容器对象
	containerObj := &Container{
		ID:      resp.ID,
		Name:    config.Name,
		Image:   containerConfig.Image,
		Status:  "created",
		State:   "created",
		Created: time.Now(),
		Labels:  config.Labels,
		Config:  config,
	}
	
	// 添加到容器池
	dm.containerPool[resp.ID] = containerObj
	
	dm.logger.Info("容器创建成功",
		zap.String("container_id", resp.ID),
		zap.String("name", config.Name),
		zap.String("image", containerConfig.Image))
	
	return containerObj, nil
}

// StartContainer 启动容器
func (dm *dockerManager) StartContainer(ctx context.Context, containerID string) error {
	err := dm.client.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	if err != nil {
		return fmt.Errorf("启动容器失败: %v", err)
	}
	
	// 更新容器状态
	dm.mu.Lock()
	if container, exists := dm.containerPool[containerID]; exists {
		container.Status = "running"
		container.State = "running"
		now := time.Now()
		container.Started = &now
	}
	dm.mu.Unlock()
	
	dm.logger.Info("容器启动成功", zap.String("container_id", containerID))
	return nil
}

// StopContainer 停止容器
func (dm *dockerManager) StopContainer(ctx context.Context, containerID string, timeout time.Duration) error {
	timeoutSeconds := int(timeout.Seconds())
	
	err := dm.client.ContainerStop(ctx, containerID, container.StopOptions{
		Timeout: &timeoutSeconds,
	})
	if err != nil {
		return fmt.Errorf("停止容器失败: %v", err)
	}
	
	// 更新容器状态
	dm.mu.Lock()
	if container, exists := dm.containerPool[containerID]; exists {
		container.Status = "exited"
		container.State = "exited"
		now := time.Now()
		container.Finished = &now
	}
	dm.mu.Unlock()
	
	dm.logger.Info("容器停止成功", zap.String("container_id", containerID))
	return nil
}

// RemoveContainer 删除容器
func (dm *dockerManager) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	err := dm.client.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
		Force: force,
	})
	if err != nil {
		return fmt.Errorf("删除容器失败: %v", err)
	}
	
	// 从容器池中移除
	dm.mu.Lock()
	delete(dm.containerPool, containerID)
	dm.mu.Unlock()
	
	dm.logger.Info("容器删除成功", zap.String("container_id", containerID))
	return nil
}

// RestartContainer 重启容器
func (dm *dockerManager) RestartContainer(ctx context.Context, containerID string, timeout time.Duration) error {
	timeoutSeconds := int(timeout.Seconds())
	
	err := dm.client.ContainerRestart(ctx, containerID, container.StopOptions{
		Timeout: &timeoutSeconds,
	})
	if err != nil {
		return fmt.Errorf("重启容器失败: %v", err)
	}
	
	// 更新容器状态
	dm.mu.Lock()
	if container, exists := dm.containerPool[containerID]; exists {
		container.Status = "running"
		container.State = "running"
		now := time.Now()
		container.Started = &now
	}
	dm.mu.Unlock()
	
	dm.logger.Info("容器重启成功", zap.String("container_id", containerID))
	return nil
}

// GetContainer 获取容器信息
func (dm *dockerManager) GetContainer(ctx context.Context, containerID string) (*Container, error) {
	// 首先从缓存中查找
	dm.mu.RLock()
	if container, exists := dm.containerPool[containerID]; exists {
		dm.mu.RUnlock()
		return container, nil
	}
	dm.mu.RUnlock()
	
	// 从Docker API获取容器信息
	inspect, err := dm.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("获取容器信息失败: %v", err)
	}
	
	container := dm.inspectToContainer(&inspect)
	
	// 更新缓存
	dm.mu.Lock()
	dm.containerPool[containerID] = container
	dm.mu.Unlock()
	
	return container, nil
}

// ListContainers 列出容器
func (dm *dockerManager) ListContainers(ctx context.Context, filter *ContainerFilter) ([]*Container, error) {
	options := types.ContainerListOptions{
		All: true,
	}
	
	// 应用过滤器
	if filter != nil {
		filters := make(map[string][]string)
		
		if len(filter.Status) > 0 {
			filters["status"] = filter.Status
		}
		
		if len(filter.Names) > 0 {
			filters["name"] = filter.Names
		}
		
		for k, v := range filter.Labels {
			filters["label"] = append(filters["label"], fmt.Sprintf("%s=%s", k, v))
		}
		
		if filter.Limit > 0 {
			options.Limit = filter.Limit
		}
		
		if filter.Since != "" {
			options.Since = filter.Since
		}
		
		if filter.Before != "" {
			options.Before = filter.Before
		}
	}
	
	containers, err := dm.client.ContainerList(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("列出容器失败: %v", err)
	}
	
	result := make([]*Container, 0, len(containers))
	for _, c := range containers {
		container := dm.containerToContainer(&c)
		result = append(result, container)
	}
	
	return result, nil
}

// GetContainerLogs 获取容器日志
func (dm *dockerManager) GetContainerLogs(ctx context.Context, containerID string, options *LogOptions) (io.ReadCloser, error) {
	logOptions := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	}
	
	if options != nil {
		logOptions.ShowStdout = options.ShowStdout
		logOptions.ShowStderr = options.ShowStderr
		logOptions.Follow = options.Follow
		logOptions.Timestamps = options.Timestamps
		logOptions.Tail = options.Tail
		
		if !options.Since.IsZero() {
			logOptions.Since = options.Since.Format(time.RFC3339)
		}
		
		if !options.Until.IsZero() {
			logOptions.Until = options.Until.Format(time.RFC3339)
		}
	}
	
	logs, err := dm.client.ContainerLogs(ctx, containerID, logOptions)
	if err != nil {
		return nil, fmt.Errorf("获取容器日志失败: %v", err)
	}
	
	return logs, nil
}

// GetContainerStats 获取容器统计信息
func (dm *dockerManager) GetContainerStats(ctx context.Context, containerID string) (*ContainerStats, error) {
	stats, err := dm.client.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, fmt.Errorf("获取容器统计信息失败: %v", err)
	}
	defer stats.Body.Close()
	
	var dockerStats types.Stats
	if err := json.NewDecoder(stats.Body).Decode(&dockerStats); err != nil {
		return nil, fmt.Errorf("解析统计信息失败: %v", err)
	}
	
	containerStats := &ContainerStats{
		ContainerID: containerID,
		Timestamp:   time.Now(),
	}
	
	// 计算CPU使用率
	if dockerStats.CPUStats.SystemUsage > 0 && dockerStats.PreCPUStats.SystemUsage > 0 {
		cpuDelta := float64(dockerStats.CPUStats.CPUUsage.TotalUsage - dockerStats.PreCPUStats.CPUUsage.TotalUsage)
		systemDelta := float64(dockerStats.CPUStats.SystemUsage - dockerStats.PreCPUStats.SystemUsage)
		onlineCPUs := float64(dockerStats.CPUStats.OnlineCPUs)
		
		if systemDelta > 0 && onlineCPUs > 0 {
			containerStats.CPUUsage = (cpuDelta / systemDelta) * onlineCPUs * 100.0
		}
	}
	
	// 内存统计
	containerStats.MemoryUsage = int64(dockerStats.MemoryStats.Usage)
	containerStats.MemoryLimit = int64(dockerStats.MemoryStats.Limit)
	
	// 网络统计
	for _, network := range dockerStats.Networks {
		containerStats.NetworkRx += int64(network.RxBytes)
		containerStats.NetworkTx += int64(network.TxBytes)
	}
	
	// 磁盘统计
	for _, ioStat := range dockerStats.BlkioStats.IoServiceBytesRecursive {
		if ioStat.Op == "read" {
			containerStats.DiskRead += int64(ioStat.Value)
		} else if ioStat.Op == "write" {
			containerStats.DiskWrite += int64(ioStat.Value)
		}
	}
	
	return containerStats, nil
}

// PullImage 拉取镜像
func (dm *dockerManager) PullImage(ctx context.Context, imageName string) error {
	reader, err := dm.client.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("拉取镜像失败: %v", err)
	}
	defer reader.Close()
	
	// 读取拉取进度（这里简化处理，实际可以解析进度信息）
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("读取拉取进度失败: %v", err)
	}
	
	dm.logger.Info("镜像拉取成功", zap.String("image", imageName))
	return nil
}

// BuildImage 构建镜像
func (dm *dockerManager) BuildImage(ctx context.Context, buildContext io.Reader, options *BuildOptions) (string, error) {
	buildOpts := types.ImageBuildOptions{
		Tags: options.Tags,
		NoCache: options.NoCache,
		PullParent: options.PullParent,
		BuildArgs: options.BuildArgs,
		Labels: options.Labels,
		Target: options.Target,
	}
	
	if options.Dockerfile != "" {
		buildOpts.Dockerfile = options.Dockerfile
	}
	
	response, err := dm.client.ImageBuild(ctx, buildContext, buildOpts)
	if err != nil {
		return "", fmt.Errorf("构建镜像失败: %v", err)
	}
	defer response.Body.Close()
	
	// 读取构建输出
	buildOutput, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("读取构建输出失败: %v", err)
	}
	
	dm.logger.Info("镜像构建成功", zap.Strings("tags", options.Tags))
	
	// 返回构建输出（实际应该解析并返回镜像ID）
	return string(buildOutput), nil
}

// 辅助方法

// inspectToContainer 将Docker检查结果转换为Container对象
func (dm *dockerManager) inspectToContainer(inspect *types.ContainerJSON) *Container {
	container := &Container{
		ID:      inspect.ID,
		Name:    strings.TrimPrefix(inspect.Name, "/"),
		Image:   inspect.Config.Image,
		Status:  inspect.State.Status,
		State:   inspect.State.Status,
		Created: inspect.Created,
		Labels:  inspect.Config.Labels,
	}
	
	if inspect.State.StartedAt != "" {
		if startTime, err := time.Parse(time.RFC3339Nano, inspect.State.StartedAt); err == nil {
			container.Started = &startTime
		}
	}
	
	if inspect.State.FinishedAt != "" {
		if finishTime, err := time.Parse(time.RFC3339Nano, inspect.State.FinishedAt); err == nil {
			container.Finished = &finishTime
		}
	}
	
	if inspect.State.ExitCode != 0 {
		container.ExitCode = &inspect.State.ExitCode
	}
	
	// 转换端口绑定
	for port, bindings := range inspect.NetworkSettings.Ports {
		for _, binding := range bindings {
			container.Ports = append(container.Ports, PortBinding{
				ContainerPort: port.Port(),
				HostPort:      binding.HostPort,
				Protocol:      port.Proto(),
			})
		}
	}
	
	// 转换网络附加信息
	for networkName, network := range inspect.NetworkSettings.Networks {
		container.Networks = append(container.Networks, NetworkAttachment{
			NetworkID:   network.NetworkID,
			NetworkName: networkName,
			IPAddress:   network.IPAddress,
			Gateway:     network.Gateway,
		})
	}
	
	return container
}

// containerToContainer 将Docker容器列表项转换为Container对象
func (dm *dockerManager) containerToContainer(c *types.Container) *Container {
	container := &Container{
		ID:      c.ID,
		Image:   c.Image,
		Status:  c.Status,
		State:   c.State,
		Created: time.Unix(c.Created, 0),
		Labels:  c.Labels,
	}
	
	if len(c.Names) > 0 {
		container.Name = strings.TrimPrefix(c.Names[0], "/")
	}
	
	// 转换端口信息
	for _, port := range c.Ports {
		container.Ports = append(container.Ports, PortBinding{
			ContainerPort: strconv.Itoa(int(port.PrivatePort)),
			HostPort:      strconv.Itoa(int(port.PublicPort)),
			Protocol:      port.Type,
		})
	}
	
	return container
}
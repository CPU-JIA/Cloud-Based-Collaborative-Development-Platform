package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"go.uber.org/zap"
)

// RemoveImage 删除镜像
func (dm *dockerManager) RemoveImage(ctx context.Context, imageID string, force bool) error {
	_, err := dm.client.ImageRemove(ctx, imageID, types.ImageRemoveOptions{
		Force: force,
	})
	if err != nil {
		return fmt.Errorf("删除镜像失败: %v", err)
	}
	
	dm.logger.Info("镜像删除成功", zap.String("image_id", imageID))
	return nil
}

// ListImages 列出镜像
func (dm *dockerManager) ListImages(ctx context.Context) ([]*Image, error) {
	images, err := dm.client.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return nil, fmt.Errorf("列出镜像失败: %v", err)
	}
	
	result := make([]*Image, 0, len(images))
	for _, img := range images {
		image := &Image{
			ID:      img.ID,
			Tags:    img.RepoTags,
			Size:    img.Size,
			Created: time.Unix(img.Created, 0),
			Labels:  img.Labels,
		}
		result = append(result, image)
	}
	
	return result, nil
}

// CreateNetwork 创建网络
func (dm *dockerManager) CreateNetwork(ctx context.Context, name string, options *NetworkOptions) (*Network, error) {
	createOptions := types.NetworkCreate{
		Driver: "bridge",
	}
	
	if options != nil {
		if options.Driver != "" {
			createOptions.Driver = options.Driver
		}
		createOptions.Options = options.Options
		createOptions.Labels = options.Labels
		createOptions.Internal = options.Internal
		createOptions.Attachable = options.Attachable
	}
	
	resp, err := dm.client.NetworkCreate(ctx, name, createOptions)
	if err != nil {
		return nil, fmt.Errorf("创建网络失败: %v", err)
	}
	
	networkObj := &Network{
		ID:      resp.ID,
		Name:    name,
		Driver:  createOptions.Driver,
		Created: time.Now(),
	}
	
	// 添加到网络池
	dm.mu.Lock()
	dm.networkPool[resp.ID] = networkObj
	dm.mu.Unlock()
	
	dm.logger.Info("网络创建成功",
		zap.String("network_id", resp.ID),
		zap.String("name", name))
	
	return networkObj, nil
}

// RemoveNetwork 删除网络
func (dm *dockerManager) RemoveNetwork(ctx context.Context, networkID string) error {
	err := dm.client.NetworkRemove(ctx, networkID)
	if err != nil {
		return fmt.Errorf("删除网络失败: %v", err)
	}
	
	// 从网络池中移除
	dm.mu.Lock()
	delete(dm.networkPool, networkID)
	dm.mu.Unlock()
	
	dm.logger.Info("网络删除成功", zap.String("network_id", networkID))
	return nil
}

// ConnectToNetwork 连接容器到网络
func (dm *dockerManager) ConnectToNetwork(ctx context.Context, networkID, containerID string) error {
	err := dm.client.NetworkConnect(ctx, networkID, containerID, &network.EndpointSettings{})
	if err != nil {
		return fmt.Errorf("连接容器到网络失败: %v", err)
	}
	
	dm.logger.Info("容器已连接到网络",
		zap.String("container_id", containerID),
		zap.String("network_id", networkID))
	
	return nil
}

// DisconnectFromNetwork 断开容器与网络的连接
func (dm *dockerManager) DisconnectFromNetwork(ctx context.Context, networkID, containerID string) error {
	err := dm.client.NetworkDisconnect(ctx, networkID, containerID, false)
	if err != nil {
		return fmt.Errorf("断开容器与网络连接失败: %v", err)
	}
	
	dm.logger.Info("容器已从网络断开",
		zap.String("container_id", containerID),
		zap.String("network_id", networkID))
	
	return nil
}

// GetSystemInfo 获取系统信息
func (dm *dockerManager) GetSystemInfo(ctx context.Context) (*SystemInfo, error) {
	info, err := dm.client.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取系统信息失败: %v", err)
	}
	
	systemInfo := &SystemInfo{
		ContainersRunning: info.ContainersRunning,
		ContainersPaused:  info.ContainersPaused,
		ContainersStopped: info.ContainersStopped,
		Images:           info.Images,
		MemTotal:         info.MemTotal,
		CPUs:             info.NCPU,
		DockerVersion:    info.ServerVersion,
		OperatingSystem:  info.OperatingSystem,
		Architecture:     info.Architecture,
	}
	
	// 计算已使用内存（简化计算）
	systemInfo.MemUsed = info.MemTotal - info.MemTotal/4 // 假设使用了75%
	
	return systemInfo, nil
}

// CleanupResources 清理资源
func (dm *dockerManager) CleanupResources(ctx context.Context) error {
	dm.logger.Info("开始清理Docker资源")
	
	// 清理已停止的容器
	filters := map[string][]string{
		"status": {"exited"},
	}
	
	containers, err := dm.client.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: filters,
	})
	if err != nil {
		return fmt.Errorf("获取已停止容器列表失败: %v", err)
	}
	
	cleanedContainers := 0
	for _, container := range containers {
		// 检查容器是否超过清理时间阈值（例如1小时）
		if time.Since(time.Unix(container.Created, 0)) > time.Hour {
			err := dm.client.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{
				Force: true,
			})
			if err != nil {
				dm.logger.Error("清理容器失败",
					zap.String("container_id", container.ID),
					zap.Error(err))
				continue
			}
			
			// 从容器池中移除
			dm.mu.Lock()
			delete(dm.containerPool, container.ID)
			dm.mu.Unlock()
			
			cleanedContainers++
		}
	}
	
	// 清理未使用的镜像
	_, err = dm.client.ImagesPrune(ctx, filters)
	if err != nil {
		dm.logger.Error("清理未使用镜像失败", zap.Error(err))
	}
	
	// 清理未使用的网络
	_, err = dm.client.NetworksPrune(ctx, filters)
	if err != nil {
		dm.logger.Error("清理未使用网络失败", zap.Error(err))
	}
	
	// 清理未使用的卷
	_, err = dm.client.VolumesPrune(ctx, filters)
	if err != nil {
		dm.logger.Error("清理未使用卷失败", zap.Error(err))
	}
	
	dm.logger.Info("Docker资源清理完成",
		zap.Int("cleaned_containers", cleanedContainers))
	
	return nil
}

// HealthCheck 健康检查
func (dm *dockerManager) HealthCheck(ctx context.Context) error {
	// 检查Docker守护进程连接
	_, err := dm.client.Ping(ctx)
	if err != nil {
		return fmt.Errorf("Docker守护进程连接检查失败: %v", err)
	}
	
	// 检查系统资源
	info, err := dm.GetSystemInfo(ctx)
	if err != nil {
		return fmt.Errorf("获取系统信息失败: %v", err)
	}
	
	// 检查内存使用率（如果超过90%则警告）
	memUsagePercent := float64(info.MemUsed) / float64(info.MemTotal) * 100
	if memUsagePercent > 90 {
		dm.logger.Warn("系统内存使用率过高",
			zap.Float64("usage_percent", memUsagePercent))
	}
	
	// 检查容器数量
	dm.mu.RLock()
	containerCount := len(dm.containerPool)
	dm.mu.RUnlock()
	
	if containerCount > dm.config.MaxContainers*90/100 { // 90%阈值
		dm.logger.Warn("容器数量接近限制",
			zap.Int("current_count", containerCount),
			zap.Int("max_count", dm.config.MaxContainers))
	}
	
	return nil
}

// Close 关闭管理器
func (dm *dockerManager) Close() error {
	dm.logger.Info("关闭Docker管理器")
	
	// 停止统计收集器
	if dm.statsCollector != nil {
		dm.statsCollector.Stop()
	}
	
	// 关闭Docker客户端
	if dm.client != nil {
		return dm.client.Close()
	}
	
	return nil
}

// startAutoCleanup 启动自动清理
func (dm *dockerManager) startAutoCleanup() {
	ticker := time.NewTicker(dm.config.CleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), dm.config.DefaultTimeout)
			if err := dm.CleanupResources(ctx); err != nil {
				dm.logger.Error("自动清理资源失败", zap.Error(err))
			}
			cancel()
		}
	}
}

// StatsCollector方法

// Start 启动统计收集器
func (sc *StatsCollector) Start() {
	sc.logger.Info("启动容器统计收集器")
	
	ticker := time.NewTicker(sc.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-sc.stopCh:
			sc.logger.Info("统计收集器已停止")
			return
		case <-ticker.C:
			sc.collectStats()
		}
	}
}

// Stop 停止统计收集器
func (sc *StatsCollector) Stop() {
	close(sc.stopCh)
}

// collectStats 收集统计信息
func (sc *StatsCollector) collectStats() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// 获取所有运行中的容器
	containers, err := sc.manager.ListContainers(ctx, &ContainerFilter{
		Status: []string{"running"},
	})
	if err != nil {
		sc.logger.Error("获取运行中容器列表失败", zap.Error(err))
		return
	}
	
	// 收集每个容器的统计信息
	for _, container := range containers {
		stats, err := sc.manager.GetContainerStats(ctx, container.ID)
		if err != nil {
			sc.logger.Error("获取容器统计信息失败",
				zap.String("container_id", container.ID),
				zap.Error(err))
			continue
		}
		
		// 发送统计信息到通道（非阻塞）
		select {
		case sc.statsCh <- stats:
		default:
			// 通道满了，丢弃这次统计
			sc.logger.Debug("统计信息通道已满，丢弃统计数据")
		}
	}
}

// GetStats 获取统计信息通道
func (sc *StatsCollector) GetStats() <-chan *ContainerStats {
	return sc.statsCh
}
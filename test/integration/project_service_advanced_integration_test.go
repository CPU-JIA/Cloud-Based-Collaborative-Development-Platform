package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/models"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/service"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/webhook"
	"github.com/cloud-platform/collaborative-dev/shared/response"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestDistributedTransactionScenarios 测试分布式事务场景
func (suite *ProjectServiceIntegrationTestSuite) TestDistributedTransactionScenarios() {
	suite.Run("分布式事务成功场景", func() {
		// 1. 创建项目
		createProjectReq := models.CreateProjectRequest{
			Key:         "tx-test",
			Name:        "事务测试项目",
			Description: stringPtr("用于测试分布式事务"),
		}
		
		resp, body := suite.makeRequest("POST", "/api/v1/projects", createProjectReq, suite.authToken)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		var createProjectResponse response.Response
		err := json.Unmarshal(body, &createProjectResponse)
		assert.NoError(suite.T(), err)
		
		projectData, err := json.Marshal(createProjectResponse.Data)
		assert.NoError(suite.T(), err)
		
		var project models.Project
		err = json.Unmarshal(projectData, &project)
		assert.NoError(suite.T(), err)
		
		projectID := project.ID
		
		// 2. 创建仓库（这将触发分布式事务）
		createRepoReq := service.CreateRepositoryRequest{
			Name:        "tx-test-repo",
			Description: stringPtr("事务测试仓库"),
			Visibility:  "private",
			InitReadme:  true,
		}
		
		// 清空调用日志
		suite.gitClient.ClearCallLog()
		
		resp, body = suite.makeRequest("POST", fmt.Sprintf("/api/v1/projects/%s/repositories", projectID), createRepoReq, suite.authToken)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		var createRepoResponse response.Response
		err = json.Unmarshal(body, &createRepoResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 200, createRepoResponse.Code)
		
		// 验证Git网关被正确调用
		callLog := suite.gitClient.GetCallLog()
		assert.Contains(suite.T(), callLog, "CreateRepository: tx-test-repo")
		
		// 验证仓库在Git网关中创建成功
		repoData, err := json.Marshal(createRepoResponse.Data)
		assert.NoError(suite.T(), err)
		
		var repository models.Repository
		err = json.Unmarshal(repoData, &repository)
		assert.NoError(suite.T(), err)
		
		// 验证仓库确实在Mock客户端中存在
		gitRepo, err := suite.gitClient.GetRepository(context.Background(), repository.ID)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), "tx-test-repo", gitRepo.Name)
	})
	
	suite.Run("分布式事务回滚场景", func() {
		// 1. 创建项目
		createProjectReq := models.CreateProjectRequest{
			Key:         "rollback-test",
			Name:        "回滚测试项目",
			Description: stringPtr("用于测试事务回滚"),
		}
		
		resp, body := suite.makeRequest("POST", "/api/v1/projects", createProjectReq, suite.authToken)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		var createProjectResponse response.Response
		err := json.Unmarshal(body, &createProjectResponse)
		assert.NoError(suite.T(), err)
		
		projectData, err := json.Marshal(createProjectResponse.Data)
		assert.NoError(suite.T(), err)
		
		var project models.Project
		err = json.Unmarshal(projectData, &project)
		assert.NoError(suite.T(), err)
		
		projectID := project.ID
		
		// 2. 设置Git网关创建仓库时出错
		suite.gitClient.SetError("CreateRepository", fmt.Errorf("Git网关服务不可用"))
		
		// 3. 尝试创建仓库（应该失败并触发回滚）
		createRepoReq := service.CreateRepositoryRequest{
			Name:        "rollback-repo",
			Description: stringPtr("回滚测试仓库"),
			Visibility:  "private",
			InitReadme:  true,
		}
		
		resp, body = suite.makeRequest("POST", fmt.Sprintf("/api/v1/projects/%s/repositories", projectID), createRepoReq, suite.authToken)
		assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)
		
		var errorResponse response.Response
		err = json.Unmarshal(body, &errorResponse)
		assert.NoError(suite.T(), err)
		assert.NotEqual(suite.T(), 200, errorResponse.Code)
		assert.Contains(suite.T(), errorResponse.Message, "创建仓库失败")
		
		// 4. 清除错误设置
		suite.gitClient.SetError("CreateRepository", nil)
		
		// 5. 验证项目仍然存在（没有被错误地删除）
		resp, body = suite.makeRequest("GET", fmt.Sprintf("/api/v1/projects/%s", projectID), nil, suite.authToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var getProjectResponse response.Response
		err = json.Unmarshal(body, &getProjectResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 200, getProjectResponse.Code)
	})
	
	suite.Run("补偿事务场景", func() {
		// 1. 创建项目
		createProjectReq := models.CreateProjectRequest{
			Key:         "compensation-test",
			Name:        "补偿测试项目",
			Description: stringPtr("用于测试补偿事务"),
		}
		
		resp, body := suite.makeRequest("POST", "/api/v1/projects", createProjectReq, suite.authToken)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		var createProjectResponse response.Response
		err := json.Unmarshal(body, &createProjectResponse)
		assert.NoError(suite.T(), err)
		
		projectData, err := json.Marshal(createProjectResponse.Data)
		assert.NoError(suite.T(), err)
		
		var project models.Project
		err = json.Unmarshal(projectData, &project)
		assert.NoError(suite.T(), err)
		
		projectID := project.ID
		
		// 2. 创建仓库成功
		createRepoReq := service.CreateRepositoryRequest{
			Name:        "compensation-repo",
			Description: stringPtr("补偿测试仓库"),
			Visibility:  "private",
			InitReadme:  true,
		}
		
		resp, body = suite.makeRequest("POST", fmt.Sprintf("/api/v1/projects/%s/repositories", projectID), createRepoReq, suite.authToken)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		var createRepoResponse response.Response
		err = json.Unmarshal(body, &createRepoResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 200, createRepoResponse.Code)
		
		repoData, err := json.Marshal(createRepoResponse.Data)
		assert.NoError(suite.T(), err)
		
		var repository models.Repository
		err = json.Unmarshal(repoData, &repository)
		assert.NoError(suite.T(), err)
		
		repositoryID := repository.ID
		
		// 3. 设置Git网关删除仓库时出错，然后尝试删除仓库
		suite.gitClient.SetError("DeleteRepository", fmt.Errorf("删除操作失败"))
		
		resp, body = suite.makeRequest("DELETE", fmt.Sprintf("/api/v1/repositories/%s", repositoryID), nil, suite.authToken)
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.StatusCode)
		
		// 4. 清除错误设置并验证补偿机制
		suite.gitClient.SetError("DeleteRepository", nil)
		
		// 验证仓库仍然存在（删除失败，没有触发补偿）
		resp, body = suite.makeRequest("GET", fmt.Sprintf("/api/v1/repositories/%s", repositoryID), nil, suite.authToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
	})
}

// TestErrorHandlingScenarios 测试错误处理场景
func (suite *ProjectServiceIntegrationTestSuite) TestErrorHandlingScenarios() {
	suite.Run("权限验证错误", func() {
		// 1. 创建项目（使用管理员账户）
		createReq := models.CreateProjectRequest{
			Key:         "permission-test",
			Name:        "权限测试项目",
			Description: stringPtr("用于测试权限控制"),
		}
		
		resp, body := suite.makeRequest("POST", "/api/v1/projects", createReq, suite.adminAuthToken)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		var createResponse response.Response
		err := json.Unmarshal(body, &createResponse)
		assert.NoError(suite.T(), err)
		
		projectData, err := json.Marshal(createResponse.Data)
		assert.NoError(suite.T(), err)
		
		var project models.Project
		err = json.Unmarshal(projectData, &project)
		assert.NoError(suite.T(), err)
		
		projectID := project.ID
		
		// 2. 尝试用普通用户更新项目（应该失败）
		updateReq := models.UpdateProjectRequest{
			Name: stringPtr("尝试更新的项目名"),
		}
		
		resp, body = suite.makeRequest("PUT", fmt.Sprintf("/api/v1/projects/%s", projectID), updateReq, suite.memberAuthToken)
		assert.Equal(suite.T(), http.StatusForbidden, resp.StatusCode)
		
		var errorResponse response.Response
		err = json.Unmarshal(body, &errorResponse)
		assert.NoError(suite.T(), err)
		assert.NotEqual(suite.T(), 200, errorResponse.Code)
		assert.Contains(suite.T(), errorResponse.Message, "无权限更新此项目")
		
		// 3. 尝试用普通用户删除项目（应该失败）
		resp, body = suite.makeRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s", projectID), nil, suite.memberAuthToken)
		assert.Equal(suite.T(), http.StatusForbidden, resp.StatusCode)
		
		err = json.Unmarshal(body, &errorResponse)
		assert.NoError(suite.T(), err)
		assert.NotEqual(suite.T(), 200, errorResponse.Code)
		assert.Contains(suite.T(), errorResponse.Message, "无权限删除此项目")
	})
	
	suite.Run("资源不存在错误", func() {
		// 1. 尝试获取不存在的项目
		nonExistentID := uuid.New()
		resp, body := suite.makeRequest("GET", fmt.Sprintf("/api/v1/projects/%s", nonExistentID), nil, suite.authToken)
		assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)
		
		var errorResponse response.Response
		err := json.Unmarshal(body, &errorResponse)
		assert.NoError(suite.T(), err)
		assert.NotEqual(suite.T(), 200, errorResponse.Code)
		
		// 2. 尝试获取不存在的仓库
		resp, body = suite.makeRequest("GET", fmt.Sprintf("/api/v1/repositories/%s", nonExistentID), nil, suite.authToken)
		assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)
		
		err = json.Unmarshal(body, &errorResponse)
		assert.NoError(suite.T(), err)
		assert.NotEqual(suite.T(), 200, errorResponse.Code)
	})
	
	suite.Run("参数验证错误", func() {
		// 1. 创建项目时使用无效的key
		createReq := models.CreateProjectRequest{
			Key:         "a", // 太短
			Name:        "测试项目",
			Description: stringPtr("测试描述"),
		}
		
		resp, body := suite.makeRequest("POST", "/api/v1/projects", createReq, suite.authToken)
		assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)
		
		var errorResponse response.Response
		err := json.Unmarshal(body, &errorResponse)
		assert.NoError(suite.T(), err)
		assert.NotEqual(suite.T(), 200, errorResponse.Code)
		
		// 2. 创建项目时使用无效格式的key
		createReq.Key = "123-invalid-start" // 不能以数字开头
		
		resp, body = suite.makeRequest("POST", "/api/v1/projects", createReq, suite.authToken)
		assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)
		
		err = json.Unmarshal(body, &errorResponse)
		assert.NoError(suite.T(), err)
		assert.NotEqual(suite.T(), 200, errorResponse.Code)
		assert.Contains(suite.T(), errorResponse.Message, "项目key只能包含字母、数字和连字符，且必须以字母开头")
	})
	
	suite.Run("Git网关服务错误", func() {
		// 1. 创建项目
		createProjectReq := models.CreateProjectRequest{
			Key:         "git-error-test",
			Name:        "Git错误测试项目",
			Description: stringPtr("用于测试Git网关错误"),
		}
		
		resp, body := suite.makeRequest("POST", "/api/v1/projects", createProjectReq, suite.authToken)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		var createProjectResponse response.Response
		err := json.Unmarshal(body, &createProjectResponse)
		assert.NoError(suite.T(), err)
		
		projectData, err := json.Marshal(createProjectResponse.Data)
		assert.NoError(suite.T(), err)
		
		var project models.Project
		err = json.Unmarshal(projectData, &project)
		assert.NoError(suite.T(), err)
		
		projectID := project.ID
		
		// 2. 设置Git网关获取仓库列表错误
		suite.gitClient.SetError("ListRepositories", fmt.Errorf("Git网关连接超时"))
		
		// 3. 尝试获取仓库列表
		resp, body = suite.makeRequest("GET", fmt.Sprintf("/api/v1/projects/%s/repositories", projectID), nil, suite.authToken)
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.StatusCode)
		
		var errorResponse response.Response
		err = json.Unmarshal(body, &errorResponse)
		assert.NoError(suite.T(), err)
		assert.NotEqual(suite.T(), 200, errorResponse.Code)
		assert.Contains(suite.T(), errorResponse.Message, "获取仓库列表失败")
		
		// 4. 清除错误设置
		suite.gitClient.SetError("ListRepositories", nil)
	})
}

// TestWebhookHandling 测试Webhook处理
func (suite *ProjectServiceIntegrationTestSuite) TestWebhookHandling() {
	suite.Run("Git Webhook处理", func() {
		// 1. 测试Webhook健康检查
		resp, body := suite.makeRequest("GET", "/api/v1/webhooks/health", nil, "")
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var healthResponse response.Response
		err := json.Unmarshal(body, &healthResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 200, healthResponse.Code)
		
		// 2. 创建测试项目和仓库
		createProjectReq := models.CreateProjectRequest{
			Key:         "webhook-test",
			Name:        "Webhook测试项目",
			Description: stringPtr("用于测试Webhook处理"),
		}
		
		resp, body = suite.makeRequest("POST", "/api/v1/projects", createProjectReq, suite.authToken)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		var createProjectResponse response.Response
		err = json.Unmarshal(body, &createProjectResponse)
		assert.NoError(suite.T(), err)
		
		projectData, err := json.Marshal(createProjectResponse.Data)
		assert.NoError(suite.T(), err)
		
		var project models.Project
		err = json.Unmarshal(projectData, &project)
		assert.NoError(suite.T(), err)
		
		projectID := project.ID
		
		// 创建仓库
		createRepoReq := service.CreateRepositoryRequest{
			Name:        "webhook-repo",
			Description: stringPtr("Webhook测试仓库"),
			Visibility:  "private",
			InitReadme:  true,
		}
		
		resp, body = suite.makeRequest("POST", fmt.Sprintf("/api/v1/projects/%s/repositories", projectID), createRepoReq, suite.authToken)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		var createRepoResponse response.Response
		err = json.Unmarshal(body, &createRepoResponse)
		assert.NoError(suite.T(), err)
		
		repoData, err := json.Marshal(createRepoResponse.Data)
		assert.NoError(suite.T(), err)
		
		var repository models.Repository
		err = json.Unmarshal(repoData, &repository)
		assert.NoError(suite.T(), err)
		
		// 3. 模拟Git Webhook事件
		webhookEvent := webhook.GitEvent{
			EventType:    "push",
			RepositoryID: repository.ID.String(),
			ProjectID:    projectID.String(),
			UserID:       suite.testUserID.String(),
			Timestamp:    time.Now(),
		}
		
		resp, body = suite.makeRequest("POST", "/api/v1/webhooks/git", webhookEvent, "")
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var webhookResponse response.Response
		err = json.Unmarshal(body, &webhookResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 200, webhookResponse.Code)
	})
}

// TestConcurrentOperations 测试并发操作
func (suite *ProjectServiceIntegrationTestSuite) TestConcurrentOperations() {
	suite.Run("并发创建项目", func() {
		const numGoroutines = 5
		results := make(chan struct {
			success bool
			error   error
		}, numGoroutines)
		
		// 并发创建多个项目
		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				createReq := models.CreateProjectRequest{
					Key:         fmt.Sprintf("concurrent-test-%d", index),
					Name:        fmt.Sprintf("并发测试项目%d", index),
					Description: stringPtr(fmt.Sprintf("并发测试项目描述%d", index)),
				}
				
				resp, body := suite.makeRequest("POST", "/api/v1/projects", createReq, suite.authToken)
				
				var createResponse response.Response
				err := json.Unmarshal(body, &createResponse)
				
				results <- struct {
					success bool
					error   error
				}{
					success: resp.StatusCode == http.StatusCreated && createResponse.Code == 200,
					error:   err,
				}
			}(i)
		}
		
		// 收集结果
		successCount := 0
		for i := 0; i < numGoroutines; i++ {
			result := <-results
			assert.NoError(suite.T(), result.error)
			if result.success {
				successCount++
			}
		}
		
		// 所有项目都应该创建成功
		assert.Equal(suite.T(), numGoroutines, successCount)
	})
	
	suite.Run("并发仓库操作", func() {
		// 1. 创建项目
		createProjectReq := models.CreateProjectRequest{
			Key:         "concurrent-repo-test",
			Name:        "并发仓库测试项目",
			Description: stringPtr("用于测试并发仓库操作"),
		}
		
		resp, body := suite.makeRequest("POST", "/api/v1/projects", createProjectReq, suite.authToken)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		var createProjectResponse response.Response
		err := json.Unmarshal(body, &createProjectResponse)
		assert.NoError(suite.T(), err)
		
		projectData, err := json.Marshal(createProjectResponse.Data)
		assert.NoError(suite.T(), err)
		
		var project models.Project
		err = json.Unmarshal(projectData, &project)
		assert.NoError(suite.T(), err)
		
		projectID := project.ID
		
		// 2. 并发创建多个仓库
		const numRepos = 3
		results := make(chan struct {
			success bool
			error   error
		}, numRepos)
		
		for i := 0; i < numRepos; i++ {
			go func(index int) {
				createRepoReq := service.CreateRepositoryRequest{
					Name:        fmt.Sprintf("concurrent-repo-%d", index),
					Description: stringPtr(fmt.Sprintf("并发仓库%d", index)),
					Visibility:  "private",
					InitReadme:  true,
				}
				
				resp, body := suite.makeRequest("POST", fmt.Sprintf("/api/v1/projects/%s/repositories", projectID), createRepoReq, suite.authToken)
				
				var createRepoResponse response.Response
				err := json.Unmarshal(body, &createRepoResponse)
				
				results <- struct {
					success bool
					error   error
				}{
					success: resp.StatusCode == http.StatusCreated && createRepoResponse.Code == 200,
					error:   err,
				}
			}(i)
		}
		
		// 收集结果
		successCount := 0
		for i := 0; i < numRepos; i++ {
			result := <-results
			assert.NoError(suite.T(), result.error)
			if result.success {
				successCount++
			}
		}
		
		// 所有仓库都应该创建成功
		assert.Equal(suite.T(), numRepos, successCount)
		
		// 3. 验证仓库列表
		resp, body = suite.makeRequest("GET", fmt.Sprintf("/api/v1/projects/%s/repositories", projectID), nil, suite.authToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var listRepoResponse response.Response
		err = json.Unmarshal(body, &listRepoResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 200, listRepoResponse.Code)
	})
}

// TestDataConsistency 测试数据一致性
func (suite *ProjectServiceIntegrationTestSuite) TestDataConsistency() {
	suite.Run("跨服务数据一致性", func() {
		// 1. 创建项目
		createProjectReq := models.CreateProjectRequest{
			Key:         "consistency-test",
			Name:        "一致性测试项目",
			Description: stringPtr("用于测试数据一致性"),
		}
		
		resp, body := suite.makeRequest("POST", "/api/v1/projects", createProjectReq, suite.authToken)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		var createProjectResponse response.Response
		err := json.Unmarshal(body, &createProjectResponse)
		assert.NoError(suite.T(), err)
		
		projectData, err := json.Marshal(createProjectResponse.Data)
		assert.NoError(suite.T(), err)
		
		var project models.Project
		err = json.Unmarshal(projectData, &project)
		assert.NoError(suite.T(), err)
		
		projectID := project.ID
		
		// 2. 创建仓库
		createRepoReq := service.CreateRepositoryRequest{
			Name:        "consistency-repo",
			Description: stringPtr("一致性测试仓库"),
			Visibility:  "private",
			InitReadme:  true,
		}
		
		resp, body = suite.makeRequest("POST", fmt.Sprintf("/api/v1/projects/%s/repositories", projectID), createRepoReq, suite.authToken)
		assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
		
		var createRepoResponse response.Response
		err = json.Unmarshal(body, &createRepoResponse)
		assert.NoError(suite.T(), err)
		
		repoData, err := json.Marshal(createRepoResponse.Data)
		assert.NoError(suite.T(), err)
		
		var repository models.Repository
		err = json.Unmarshal(repoData, &repository)
		assert.NoError(suite.T(), err)
		
		repositoryID := repository.ID
		
		// 3. 验证数据在Git网关和项目服务中的一致性
		// 从项目服务获取仓库信息
		resp, body = suite.makeRequest("GET", fmt.Sprintf("/api/v1/repositories/%s", repositoryID), nil, suite.authToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var getRepoResponse response.Response
		err = json.Unmarshal(body, &getRepoResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 200, getRepoResponse.Code)
		
		// 从Mock Git网关获取仓库信息
		gitRepo, err := suite.gitClient.GetRepository(context.Background(), repositoryID)
		assert.NoError(suite.T(), err)
		
		// 验证数据一致性
		repoFromService, err := json.Marshal(getRepoResponse.Data)
		assert.NoError(suite.T(), err)
		
		var serviceRepo models.Repository
		err = json.Unmarshal(repoFromService, &serviceRepo)
		assert.NoError(suite.T(), err)
		
		assert.Equal(suite.T(), gitRepo.ID, serviceRepo.ID)
		assert.Equal(suite.T(), gitRepo.Name, serviceRepo.Name)
		assert.Equal(suite.T(), string(gitRepo.Visibility), serviceRepo.Visibility)
		assert.Equal(suite.T(), gitRepo.ProjectID, serviceRepo.ProjectID)
		
		// 4. 更新仓库并验证一致性
		updateRepoReq := service.UpdateRepositoryRequest{
			Name:        stringPtr("updated-consistency-repo"),
			Description: stringPtr("更新的一致性测试仓库"),
		}
		
		resp, body = suite.makeRequest("PUT", fmt.Sprintf("/api/v1/repositories/%s", repositoryID), updateRepoReq, suite.authToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		var updateRepoResponse response.Response
		err = json.Unmarshal(body, &updateRepoResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 200, updateRepoResponse.Code)
		
		// 验证更新后的一致性
		updatedGitRepo, err := suite.gitClient.GetRepository(context.Background(), repositoryID)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), "updated-consistency-repo", updatedGitRepo.Name)
	})
}

// TestPerformanceScenarios 测试性能场景
func (suite *ProjectServiceIntegrationTestSuite) TestPerformanceScenarios() {
	suite.Run("大量项目列表查询性能", func() {
		// 1. 创建多个项目
		const numProjects = 20
		createdProjects := make([]uuid.UUID, 0, numProjects)
		
		for i := 0; i < numProjects; i++ {
			createReq := models.CreateProjectRequest{
				Key:         fmt.Sprintf("perf-test-%d", i),
				Name:        fmt.Sprintf("性能测试项目%d", i),
				Description: stringPtr(fmt.Sprintf("性能测试项目描述%d", i)),
			}
			
			resp, body := suite.makeRequest("POST", "/api/v1/projects", createReq, suite.authToken)
			assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)
			
			var createResponse response.Response
			err := json.Unmarshal(body, &createResponse)
			assert.NoError(suite.T(), err)
			
			projectData, err := json.Marshal(createResponse.Data)
			assert.NoError(suite.T(), err)
			
			var project models.Project
			err = json.Unmarshal(projectData, &project)
			assert.NoError(suite.T(), err)
			
			createdProjects = append(createdProjects, project.ID)
		}
		
		// 2. 测试项目列表查询性能
		startTime := time.Now()
		
		resp, body := suite.makeRequest("GET", "/api/v1/projects?page=1&page_size=10", nil, suite.authToken)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		
		queryDuration := time.Since(startTime)
		
		var listResponse response.Response
		err := json.Unmarshal(body, &listResponse)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 200, listResponse.Code)
		
		// 验证查询时间在合理范围内（应该小于1秒）
		assert.Less(suite.T(), queryDuration, 1*time.Second, "项目列表查询耗时过长")
		
		suite.logger.Info("项目列表查询性能测试",
			zap.Duration("query_duration", queryDuration),
			zap.Int("num_projects", numProjects))
	})
	
	suite.Run("分页查询性能", func() {
		// 测试不同页码的查询性能
		pages := []int{1, 2, 3}
		
		for _, page := range pages {
			startTime := time.Now()
			
			resp, body := suite.makeRequest("GET", fmt.Sprintf("/api/v1/projects?page=%d&page_size=5", page), nil, suite.authToken)
			assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
			
			queryDuration := time.Since(startTime)
			
			var listResponse response.Response
			err := json.Unmarshal(body, &listResponse)
			assert.NoError(suite.T(), err)
			assert.Equal(suite.T(), 200, listResponse.Code)
			
			// 验证查询时间在合理范围内
			assert.Less(suite.T(), queryDuration, 500*time.Millisecond, fmt.Sprintf("第%d页查询耗时过长", page))
			
			suite.logger.Info("分页查询性能测试",
				zap.Int("page", page),
				zap.Duration("query_duration", queryDuration))
		}
	})
}
# 开发者指南

## 概述

本指南旨在帮助开发者快速上手云协作开发平台的开发工作，包括项目架构、开发环境设置、编码规范、测试指南等。

## 项目架构

### 整体架构

```
Cloud-Based Collaborative Development Platform
├── Frontend (React + TypeScript)
│   ├── UI Components
│   ├── State Management (Redux)
│   ├── API Client
│   └── Real-time Communication
│
├── Backend Services (Go)
│   ├── API Gateway
│   ├── Project Service
│   ├── IAM Service
│   ├── Git Gateway Service
│   ├── File Service
│   ├── Team Service
│   ├── Notification Service
│   ├── CI/CD Service
│   └── Knowledge Base Service
│
├── Infrastructure
│   ├── PostgreSQL (主数据库)
│   ├── Redis (缓存/会话)
│   ├── MinIO (对象存储)
│   ├── NATS (消息队列)
│   └── Prometheus (监控)
│
└── DevOps
    ├── Docker Containers
    ├── Kubernetes Manifests
    ├── Helm Charts
    ├── CI/CD Pipelines
    └── Monitoring & Logging
```

### 技术栈

**前端:**
- React 18 + TypeScript
- Redux Toolkit + RTK Query
- Material-UI (MUI)
- React Router v6
- Socket.IO Client
- Vite (构建工具)

**后端:**
- Go 1.23
- Gin Web Framework
- GORM (ORM)
- JWT-Go (认证)
- Testify (测试)
- Cobra (CLI)

**数据存储:**
- PostgreSQL 15 (主数据库)
- Redis 7 (缓存/会话)
- MinIO (S3兼容对象存储)

**基础设施:**
- Docker & Docker Compose
- Kubernetes
- Helm 3
- NATS (消息队列)
- Prometheus + Grafana (监控)
- ELK Stack (日志)

## 开发环境设置

### 前置条件

1. **Go 1.23+**
```bash
# macOS
brew install go

# Ubuntu
sudo apt update
sudo apt install golang-go

# 验证安装
go version
```

2. **Node.js 18+**
```bash
# 使用nvm
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash
nvm install 18
nvm use 18

# 验证安装
node --version
npm --version
```

3. **Docker & Docker Compose**
```bash
# macOS
brew install docker docker-compose

# Ubuntu
sudo apt install docker.io docker-compose

# 启动Docker服务
sudo systemctl start docker
sudo usermod -aG docker $USER
```

4. **Git**
```bash
# 配置Git
git config --global user.name "Your Name"
git config --global user.email "your.email@example.com"
```

### 本地开发环境

1. **克隆项目**
```bash
git clone https://github.com/your-org/cloud-collaborative-platform.git
cd cloud-collaborative-platform
```

2. **环境配置**
```bash
# 复制环境配置
cp .env.example .env.development

# 编辑配置文件
# 修改数据库连接、Redis地址等配置
```

3. **启动基础服务**
```bash
# 启动PostgreSQL和Redis
docker-compose up -d postgres redis

# 等待服务启动
sleep 10

# 运行数据库迁移
make migrate-up
```

4. **安装依赖**
```bash
# Go模块依赖
go mod download

# 前端依赖
cd frontend
npm install
cd ..
```

5. **启动开发服务**
```bash
# 方式1: 使用Makefile (推荐)
make dev

# 方式2: 手动启动各服务
# 终端1: 启动后端服务
make run-services

# 终端2: 启动前端
cd frontend && npm run dev
```

6. **验证环境**
```bash
# 检查服务状态
curl http://localhost:8080/health
curl http://localhost:3000

# 运行测试
make test
```

## 项目结构

```
.
├── cmd/                          # 服务入口点
│   ├── project-service/
│   ├── iam-service/
│   ├── git-gateway-service/
│   └── ...
├── internal/                     # 内部包
│   ├── project/
│   │   ├── handler/             # HTTP处理器
│   │   ├── service/             # 业务逻辑
│   │   ├── repository/          # 数据访问
│   │   └── model/              # 数据模型
│   └── ...
├── shared/                       # 共享包
│   ├── auth/                    # 认证相关
│   ├── config/                  # 配置管理
│   ├── database/                # 数据库连接
│   ├── middleware/              # 中间件
│   ├── utils/                   # 工具函数
│   └── validation/              # 输入验证
├── frontend/                     # 前端代码
│   ├── src/
│   │   ├── components/          # React组件
│   │   ├── pages/              # 页面组件
│   │   ├── store/              # Redux状态管理
│   │   ├── api/                # API客户端
│   │   ├── utils/              # 工具函数
│   │   └── types/              # TypeScript类型
│   ├── public/
│   └── package.json
├── test/                         # 测试文件
│   ├── unit/                    # 单元测试
│   ├── integration/             # 集成测试
│   └── common/                  # 测试工具
├── docs/                         # 文档
├── deployments/                  # 部署配置
├── scripts/                      # 脚本文件
├── .github/                      # GitHub Actions
├── Makefile                      # 构建脚本
├── docker-compose.yml           # 开发环境
└── go.mod                       # Go模块
```

## 编码规范

### Go 编码规范

1. **命名规范**
```go
// 包名: 小写，简短，有意义
package project

// 常量: 大写字母+下划线
const MAX_RETRY_COUNT = 3

// 变量和函数: 驼峰命名
var userName string
func getUserInfo() {}

// 接口: 通常以-er结尾
type ProjectCreator interface {
    CreateProject() error
}

// 结构体: 大写开头的驼峰
type ProjectService struct {
    repo ProjectRepository
}
```

2. **错误处理**
```go
// 自定义错误类型
type ValidationError struct {
    Field   string
    Message string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation failed on field '%s': %s", e.Field, e.Message)
}

// 错误处理模式
func CreateProject(req CreateProjectRequest) (*Project, error) {
    if err := validateRequest(req); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }
    
    project, err := projectRepo.Create(req)
    if err != nil {
        return nil, fmt.Errorf("failed to create project: %w", err)
    }
    
    return project, nil
}
```

3. **结构体标签**
```go
type Project struct {
    ID          string    `json:"id" db:"id" validate:"required"`
    Name        string    `json:"name" db:"name" validate:"required,min=1,max=100"`
    Description string    `json:"description" db:"description" validate:"max=500"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}
```

4. **Context使用**
```go
func (s *ProjectService) GetProject(ctx context.Context, id string) (*Project, error) {
    // 检查context是否被取消
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    return s.repo.FindByID(ctx, id)
}
```

### TypeScript/React 编码规范

1. **组件定义**
```typescript
// 函数组件 + TypeScript
interface ProjectCardProps {
  project: Project;
  onEdit?: (project: Project) => void;
  onDelete?: (projectId: string) => void;
}

const ProjectCard: React.FC<ProjectCardProps> = ({ 
  project, 
  onEdit, 
  onDelete 
}) => {
  const handleEdit = useCallback(() => {
    onEdit?.(project);
  }, [onEdit, project]);

  return (
    <Card>
      <CardContent>
        <Typography variant="h6">{project.name}</Typography>
        <Typography variant="body2">{project.description}</Typography>
      </CardContent>
      <CardActions>
        <Button onClick={handleEdit}>编辑</Button>
      </CardActions>
    </Card>
  );
};

export default ProjectCard;
```

2. **状态管理 (Redux Toolkit)**
```typescript
// features/projects/projectsSlice.ts
interface ProjectsState {
  projects: Project[];
  loading: boolean;
  error: string | null;
}

const initialState: ProjectsState = {
  projects: [],
  loading: false,
  error: null,
};

const projectsSlice = createSlice({
  name: 'projects',
  initialState,
  reducers: {
    setLoading: (state, action) => {
      state.loading = action.payload;
    },
    setError: (state, action) => {
      state.error = action.payload;
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchProjects.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchProjects.fulfilled, (state, action) => {
        state.loading = false;
        state.projects = action.payload;
      })
      .addCase(fetchProjects.rejected, (state, action) => {
        state.loading = false;
        state.error = action.error.message || 'Failed to fetch projects';
      });
  },
});
```

3. **API客户端**
```typescript
// api/projectsApi.ts
export const projectsApi = createApi({
  reducerPath: 'projectsApi',
  baseQuery: fetchBaseQuery({
    baseUrl: '/api/v1/projects',
    prepareHeaders: (headers, { getState }) => {
      const token = (getState() as RootState).auth.token;
      if (token) {
        headers.set('authorization', `Bearer ${token}`);
      }
      return headers;
    },
  }),
  tagTypes: ['Project'],
  endpoints: (builder) => ({
    getProjects: builder.query<Project[], ProjectsQueryParams>({
      query: (params) => ({
        url: '',
        params,
      }),
      providesTags: ['Project'],
    }),
    createProject: builder.mutation<Project, CreateProjectRequest>({
      query: (body) => ({
        url: '',
        method: 'POST',
        body,
      }),
      invalidatesTags: ['Project'],
    }),
  }),
});
```

### 数据库规范

1. **表命名**
```sql
-- 表名: 复数，下划线分隔
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 索引命名: idx_表名_字段名
CREATE INDEX idx_projects_owner_id ON projects(owner_id);
CREATE INDEX idx_projects_name ON projects(name);
```

2. **外键约束**
```sql
-- 外键命名: fk_表名_引用表名
ALTER TABLE projects 
ADD CONSTRAINT fk_projects_users 
FOREIGN KEY (owner_id) REFERENCES users(id);
```

## 测试指南

### 单元测试

1. **Go测试**
```go
// internal/project/service/project_service_test.go
func TestProjectService_CreateProject(t *testing.T) {
    tests := []struct {
        name        string
        input       CreateProjectRequest
        mockSetup   func(*mocks.ProjectRepository)
        expected    *Project
        expectedErr error
    }{
        {
            name: "successful creation",
            input: CreateProjectRequest{
                Name:        "Test Project",
                Description: "Test Description",
            },
            mockSetup: func(repo *mocks.ProjectRepository) {
                repo.On("Create", mock.Anything, mock.Anything).
                    Return(&Project{ID: "test-id"}, nil)
            },
            expected: &Project{ID: "test-id"},
        },
        {
            name: "validation error",
            input: CreateProjectRequest{
                Name: "", // invalid
            },
            expectedErr: ErrInvalidInput,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            repo := &mocks.ProjectRepository{}
            if tt.mockSetup != nil {
                tt.mockSetup(repo)
            }

            service := NewProjectService(repo)
            result, err := service.CreateProject(context.Background(), tt.input)

            if tt.expectedErr != nil {
                assert.Error(t, err)
                assert.ErrorIs(t, err, tt.expectedErr)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected.ID, result.ID)
            }

            repo.AssertExpectations(t)
        })
    }
}
```

2. **React测试**
```typescript
// frontend/src/components/__tests__/ProjectCard.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { Provider } from 'react-redux';
import { store } from '../../store';
import ProjectCard from '../ProjectCard';

const mockProject: Project = {
  id: 'test-id',
  name: 'Test Project',
  description: 'Test Description',
  created_at: '2025-01-01T00:00:00Z',
  updated_at: '2025-01-01T00:00:00Z',
};

const renderWithProvider = (component: React.ReactElement) => {
  return render(
    <Provider store={store}>
      {component}
    </Provider>
  );
};

describe('ProjectCard', () => {
  test('renders project information', () => {
    renderWithProvider(<ProjectCard project={mockProject} />);
    
    expect(screen.getByText('Test Project')).toBeInTheDocument();
    expect(screen.getByText('Test Description')).toBeInTheDocument();
  });

  test('calls onEdit when edit button is clicked', () => {
    const onEdit = jest.fn();
    renderWithProvider(
      <ProjectCard project={mockProject} onEdit={onEdit} />
    );
    
    fireEvent.click(screen.getByText('编辑'));
    expect(onEdit).toHaveBeenCalledWith(mockProject);
  });
});
```

### 集成测试

```go
// test/integration/project_api_test.go
func TestProjectAPI_Integration(t *testing.T) {
    // 设置测试数据库
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)

    // 创建测试服务器
    app := setupTestApp(db)
    server := httptest.NewServer(app)
    defer server.Close()

    // 创建测试用户和认证token
    user := createTestUser(t, db)
    token := generateTestToken(user.ID)

    t.Run("create project", func(t *testing.T) {
        payload := map[string]interface{}{
            "name":        "Integration Test Project",
            "description": "Test Description",
        }

        resp, err := makeAuthenticatedRequest(
            "POST", 
            server.URL+"/api/v1/projects",
            payload,
            token,
        )
        require.NoError(t, err)
        assert.Equal(t, http.StatusCreated, resp.StatusCode)

        var result map[string]interface{}
        err = json.NewDecoder(resp.Body).Decode(&result)
        require.NoError(t, err)
        
        assert.True(t, result["success"].(bool))
        data := result["data"].(map[string]interface{})
        assert.Equal(t, "Integration Test Project", data["name"])
    })
}
```

### 测试命令

```bash
# 运行所有测试
make test

# 运行单元测试
make test-unit

# 运行集成测试
make test-integration

# 运行测试并生成覆盖率报告
make test-coverage

# 运行特定包的测试
go test ./internal/project/...

# 运行前端测试
cd frontend && npm test

# 前端测试覆盖率
cd frontend && npm run test:coverage
```

## 调试指南

### Go 服务调试

1. **使用Delve调试器**
```bash
# 安装delve
go install github.com/go-delve/delve/cmd/dlv@latest

# 启动调试模式
dlv debug ./cmd/project-service/main.go

# 在代码中设置断点
(dlv) break main.main
(dlv) break internal/project/handler.CreateProject
(dlv) continue
```

2. **日志调试**
```go
import (
    "github.com/sirupsen/logrus"
)

func CreateProject(req CreateProjectRequest) error {
    logrus.WithFields(logrus.Fields{
        "user_id": req.UserID,
        "project_name": req.Name,
    }).Info("Creating new project")
    
    // 业务逻辑
    
    logrus.WithField("project_id", project.ID).Info("Project created successfully")
    return nil
}
```

### 前端调试

1. **浏览器开发工具**
```typescript
// 在组件中添加调试信息
const ProjectCard: React.FC<ProjectCardProps> = ({ project }) => {
  console.log('ProjectCard rendered with:', project);
  
  useEffect(() => {
    console.log('Project updated:', project);
  }, [project]);

  return (
    // JSX
  );
};
```

2. **Redux DevTools**
```typescript
// store配置
export const store = configureStore({
  reducer: {
    projects: projectsReducer,
    auth: authReducer,
  },
  devTools: process.env.NODE_ENV !== 'production',
});
```

## 性能优化

### 后端性能优化

1. **数据库查询优化**
```go
// 使用预加载避免N+1查询
func (r *ProjectRepository) FindWithMembers(id string) (*Project, error) {
    var project Project
    err := r.db.Preload("Members.User").First(&project, "id = ?", id).Error
    return &project, err
}

// 使用原生SQL优化复杂查询
func (r *ProjectRepository) GetProjectStats(id string) (*ProjectStats, error) {
    var stats ProjectStats
    query := `
        SELECT 
            p.id,
            COUNT(DISTINCT f.id) as files_count,
            COUNT(DISTINCT m.id) as members_count,
            COUNT(DISTINCT c.id) as commits_count
        FROM projects p
        LEFT JOIN files f ON f.project_id = p.id
        LEFT JOIN project_members m ON m.project_id = p.id
        LEFT JOIN commits c ON c.project_id = p.id
        WHERE p.id = ?
        GROUP BY p.id
    `
    err := r.db.Raw(query, id).Scan(&stats).Error
    return &stats, err
}
```

2. **缓存策略**
```go
func (s *ProjectService) GetProject(ctx context.Context, id string) (*Project, error) {
    // 尝试从缓存获取
    cacheKey := fmt.Sprintf("project:%s", id)
    if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
        var project Project
        if err := json.Unmarshal(cached, &project); err == nil {
            return &project, nil
        }
    }

    // 从数据库获取
    project, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }

    // 缓存结果
    if data, err := json.Marshal(project); err == nil {
        s.cache.Set(ctx, cacheKey, data, 5*time.Minute)
    }

    return project, nil
}
```

### 前端性能优化

1. **组件优化**
```typescript
// 使用React.memo避免不必要的重渲染
const ProjectCard = React.memo<ProjectCardProps>(({ project, onEdit }) => {
  const handleEdit = useCallback(() => {
    onEdit?.(project);
  }, [onEdit, project]);

  return (
    // JSX
  );
});

// 使用useMemo缓存计算结果
const ProjectList: React.FC = () => {
  const projects = useSelector(selectProjects);
  
  const sortedProjects = useMemo(() => {
    return [...projects].sort((a, b) => 
      new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
    );
  }, [projects]);

  return (
    <div>
      {sortedProjects.map(project => (
        <ProjectCard key={project.id} project={project} />
      ))}
    </div>
  );
};
```

2. **代码分割**
```typescript
// 路由级别的代码分割
const ProjectDetails = lazy(() => import('../pages/ProjectDetails'));
const UserSettings = lazy(() => import('../pages/UserSettings'));

function App() {
  return (
    <Router>
      <Suspense fallback={<Loading />}>
        <Routes>
          <Route path="/projects/:id" element={<ProjectDetails />} />
          <Route path="/settings" element={<UserSettings />} />
        </Routes>
      </Suspense>
    </Router>
  );
}
```

## 常见问题解决

### 构建问题

1. **Go模块问题**
```bash
# 清理模块缓存
go clean -modcache

# 重新下载依赖
go mod download

# 更新依赖
go mod tidy
```

2. **前端依赖问题**
```bash
# 清理node_modules
rm -rf node_modules package-lock.json
npm install

# 清理npm缓存
npm cache clean --force
```

### 数据库问题

1. **迁移失败**
```bash
# 回滚迁移
migrate -path migrations -database $DATABASE_URL down 1

# 强制设置版本
migrate -path migrations -database $DATABASE_URL force 1

# 重新运行迁移
migrate -path migrations -database $DATABASE_URL up
```

2. **连接问题**
```bash
# 检查数据库连接
psql $DATABASE_URL -c "SELECT version();"

# 检查连接池
psql $DATABASE_URL -c "SELECT count(*) FROM pg_stat_activity;"
```

## 最佳实践

### 安全实践

1. **输入验证**
```go
func validateCreateProjectRequest(req CreateProjectRequest) error {
    if strings.TrimSpace(req.Name) == "" {
        return ErrProjectNameRequired
    }
    
    if len(req.Name) > 100 {
        return ErrProjectNameTooLong
    }
    
    // 防止XSS
    if containsHTMLTags(req.Description) {
        return ErrInvalidDescription
    }
    
    return nil
}
```

2. **权限检查**
```go
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
    projectID := c.Param("id")
    userID := getUserIDFromContext(c)
    
    // 检查用户权限
    if !h.authService.CanUpdateProject(userID, projectID) {
        c.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
        return
    }
    
    // 处理更新逻辑
}
```

### 错误处理

1. **统一错误响应**
```go
type ErrorResponse struct {
    Success bool     `json:"success"`
    Error   ApiError `json:"error"`
}

type ApiError struct {
    Code    string      `json:"code"`
    Message string      `json:"message"`
    Details interface{} `json:"details,omitempty"`
}

func handleError(c *gin.Context, err error) {
    var apiError ApiError
    
    switch {
    case errors.Is(err, ErrProjectNotFound):
        c.JSON(http.StatusNotFound, ErrorResponse{
            Success: false,
            Error: ApiError{
                Code:    "PROJECT_NOT_FOUND",
                Message: "项目不存在",
            },
        })
    case errors.Is(err, ErrValidationFailed):
        c.JSON(http.StatusBadRequest, ErrorResponse{
            Success: false,
            Error: ApiError{
                Code:    "VALIDATION_ERROR",
                Message: "输入验证失败",
                Details: err.(*ValidationError).Fields,
            },
        })
    default:
        c.JSON(http.StatusInternalServerError, ErrorResponse{
            Success: false,
            Error: ApiError{
                Code:    "INTERNAL_ERROR",
                Message: "服务器内部错误",
            },
        })
    }
}
```

## 贡献指南

### 提交代码

1. **分支策略**
```bash
# 从主分支创建功能分支
git checkout main
git pull origin main
git checkout -b feature/user-authentication

# 开发完成后提交
git add .
git commit -m "feat: add user authentication"

# 推送到远程
git push origin feature/user-authentication
```

2. **提交信息格式**
```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

类型:
- `feat`: 新功能
- `fix`: 错误修复
- `docs`: 文档更新
- `style`: 代码格式调整
- `refactor`: 代码重构
- `test`: 测试相关
- `chore`: 构建/工具相关

3. **代码审查**
- 确保所有测试通过
- 代码覆盖率不低于80%
- 通过静态代码检查
- 至少一个团队成员审查

### 发布流程

1. **版本号规则**
- 主版本号: 不兼容的API更改
- 次版本号: 向后兼容的功能性新增
- 修订号: 向后兼容的问题修复

2. **发布步骤**
```bash
# 1. 更新版本号
git tag v1.2.3

# 2. 推送标签
git push origin v1.2.3

# 3. GitHub Actions自动构建和发布
```

---

*最后更新: 2025-07-27*
*版本: 1.0.0*
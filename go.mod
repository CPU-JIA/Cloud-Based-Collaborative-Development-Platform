module github.com/cloud-platform/collaborative-dev

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/golang-jwt/jwt/v5 v5.0.0
	github.com/google/uuid v1.3.0
	github.com/lib/pq v1.10.9
	github.com/redis/go-redis/v9 v9.2.1
	github.com/spf13/cobra v1.7.0
	github.com/spf13/viper v1.17.0
	golang.org/x/crypto v0.14.0
	gorm.io/driver/postgres v1.5.4
	gorm.io/gorm v1.25.5
	
	// 配置管理
	github.com/fsnotify/fsnotify v1.7.0
	
	// 日志
	github.com/sirupsen/logrus v1.9.3
	go.uber.org/zap v1.26.0
	
	// 监控和链路追踪
	github.com/prometheus/client_golang v1.17.0
	go.opentelemetry.io/otel v1.19.0
	go.opentelemetry.io/otel/trace v1.19.0
	
	// gRPC
	google.golang.org/grpc v1.59.0
	google.golang.org/protobuf v1.31.0
	
	// 消息队列
	github.com/segmentio/kafka-go v0.4.44
	
	// 工具库
	github.com/stretchr/testify v1.8.4
	
	// Web框架相关
	github.com/gin-contrib/cors v1.4.0
	github.com/gin-contrib/sessions v0.0.5
	
	// 数据验证
	github.com/go-playground/validator/v10 v10.15.5
)

require (
	github.com/bytedance/sonic v1.9.1 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20221115062448-fe3a3abad311 // indirect
	github.com/gabriel-vasile/mimetype v1.4.2 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.14.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.2.4 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.0.8 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.11 // indirect
	golang.org/x/arch v0.3.0 // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/sys v0.11.0 // indirect
	golang.org/x/text v0.12.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
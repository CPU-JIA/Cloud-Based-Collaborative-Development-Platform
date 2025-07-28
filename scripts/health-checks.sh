#!/bin/bash
# Health checks for services

set -e

ENVIRONMENT=$1
if [ -z "$ENVIRONMENT" ]; then
    echo "Usage: $0 <environment>"
    exit 1
fi

echo "Running health checks for environment: $ENVIRONMENT"

# Service endpoints
declare -A SERVICES=(
    ["iam"]="8081"
    ["project"]="8082"
    ["file"]="8083"
    ["git-gateway"]="8084"
    ["cicd"]="8085"
    ["notification"]="8086"
    ["team"]="8087"
    ["tenant"]="8088"
)

# Get base URL based on environment
if [ "$ENVIRONMENT" = "production" ]; then
    BASE_URL="https://api.cloudplatform.com"
elif [ "$ENVIRONMENT" = "staging" ]; then
    BASE_URL="https://api-staging.cloudplatform.com"
else
    BASE_URL="http://localhost"
fi

# Health check function
health_check() {
    local service=$1
    local port=$2
    local url=""
    
    if [ "$ENVIRONMENT" = "production" ] || [ "$ENVIRONMENT" = "staging" ]; then
        url="$BASE_URL/api/v1/$service/health"
    else
        url="$BASE_URL:$port/health"
    fi
    
    echo -n "Checking $service service at $url..."
    
    response=$(curl -s -w "\n%{http_code}" "$url" 2>/dev/null || echo "000")
    status_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$status_code" = "200" ]; then
        echo " ✅ Healthy"
        
        # Parse JSON response if available
        if command -v jq &> /dev/null && [ ! -z "$body" ]; then
            echo "  Details:"
            echo "$body" | jq -r '. | to_entries[] | "    \(.key): \(.value)"' 2>/dev/null || echo "    $body"
        fi
    else
        echo " ❌ Unhealthy (Status: $status_code)"
        [ ! -z "$body" ] && echo "  Response: $body"
        return 1
    fi
}

# Kubernetes health check
k8s_health_check() {
    if ! command -v kubectl &> /dev/null; then
        echo "kubectl not found, skipping Kubernetes checks"
        return 0
    fi
    
    local namespace=""
    if [ "$ENVIRONMENT" = "production" ]; then
        namespace="production"
    elif [ "$ENVIRONMENT" = "staging" ]; then
        namespace="staging"
    else
        namespace="default"
    fi
    
    echo -e "\n=== Kubernetes Health Checks ==="
    
    # Check deployments
    echo "Checking deployments..."
    kubectl get deployments -n "$namespace" -o wide
    
    # Check pods
    echo -e "\nChecking pods..."
    kubectl get pods -n "$namespace" -o wide
    
    # Check services
    echo -e "\nChecking services..."
    kubectl get services -n "$namespace" -o wide
    
    # Check for unhealthy pods
    unhealthy=$(kubectl get pods -n "$namespace" --field-selector=status.phase!=Running,status.phase!=Succeeded -o name | wc -l)
    if [ "$unhealthy" -gt 0 ]; then
        echo -e "\n⚠️  WARNING: $unhealthy unhealthy pods detected"
        kubectl get pods -n "$namespace" --field-selector=status.phase!=Running,status.phase!=Succeeded
    fi
}

# Database health check
db_health_check() {
    echo -e "\n=== Database Health Check ==="
    
    if [ "$ENVIRONMENT" = "production" ] || [ "$ENVIRONMENT" = "staging" ]; then
        echo "Using service endpoint for database check..."
        curl -s "$BASE_URL/api/v1/health/database" | jq . 2>/dev/null || echo "Database check endpoint not available"
    else
        echo -n "Checking PostgreSQL..."
        if PGPASSWORD=${DB_PASSWORD:-postgres} psql -h ${DB_HOST:-localhost} -U ${DB_USER:-postgres} -d ${DB_NAME:-cloud_platform} -c "SELECT 1" > /dev/null 2>&1; then
            echo " ✅ Connected"
            
            # Check database stats
            echo "  Database statistics:"
            PGPASSWORD=${DB_PASSWORD:-postgres} psql -h ${DB_HOST:-localhost} -U ${DB_USER:-postgres} -d ${DB_NAME:-cloud_platform} -t -c "
                SELECT 'Active connections: ' || count(*) FROM pg_stat_activity WHERE state = 'active'
                UNION ALL
                SELECT 'Total connections: ' || count(*) FROM pg_stat_activity
                UNION ALL
                SELECT 'Database size: ' || pg_size_pretty(pg_database_size(current_database()));
            " | sed 's/^/    /'
        else
            echo " ❌ Connection failed"
        fi
    fi
}

# Redis health check
redis_health_check() {
    echo -e "\n=== Redis Health Check ==="
    
    echo -n "Checking Redis..."
    if redis-cli -h ${REDIS_HOST:-localhost} -p ${REDIS_PORT:-6379} ping > /dev/null 2>&1; then
        echo " ✅ Connected"
        
        # Get Redis info
        echo "  Redis statistics:"
        redis-cli -h ${REDIS_HOST:-localhost} -p ${REDIS_PORT:-6379} INFO server | grep -E "redis_version|uptime_in_days" | sed 's/^/    /'
        redis-cli -h ${REDIS_HOST:-localhost} -p ${REDIS_PORT:-6379} INFO memory | grep -E "used_memory_human|used_memory_peak_human" | sed 's/^/    /'
        redis-cli -h ${REDIS_HOST:-localhost} -p ${REDIS_PORT:-6379} INFO stats | grep -E "total_connections_received|total_commands_processed" | sed 's/^/    /'
    else
        echo " ❌ Connection failed"
    fi
}

# Run all health checks
echo "=== Service Health Checks ==="
failed_services=0
for service in "${!SERVICES[@]}"; do
    if ! health_check "$service" "${SERVICES[$service]}"; then
        ((failed_services++))
    fi
done

# Additional checks for production/staging
if [ "$ENVIRONMENT" = "production" ] || [ "$ENVIRONMENT" = "staging" ]; then
    k8s_health_check
fi

# Database and Redis checks
db_health_check
redis_health_check

# Summary
echo -e "\n=== Health Check Summary ==="
if [ $failed_services -eq 0 ]; then
    echo "✅ All services are healthy!"
    exit 0
else
    echo "❌ $failed_services service(s) are unhealthy!"
    exit 1
fi
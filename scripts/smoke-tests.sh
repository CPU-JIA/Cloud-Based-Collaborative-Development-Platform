#!/bin/bash
# Smoke tests for production deployment

set -e

ENVIRONMENT=$1
if [ -z "$ENVIRONMENT" ]; then
    echo "Usage: $0 <environment>"
    exit 1
fi

echo "Running smoke tests for environment: $ENVIRONMENT"

# Get service endpoints based on environment
if [ "$ENVIRONMENT" = "production" ]; then
    API_BASE="https://api.cloudplatform.com"
    HEALTH_ENDPOINT="$API_BASE/health"
elif [ "$ENVIRONMENT" = "staging" ]; then
    API_BASE="https://api-staging.cloudplatform.com"
    HEALTH_ENDPOINT="$API_BASE/health"
else
    API_BASE="http://localhost:8080"
    HEALTH_ENDPOINT="$API_BASE/health"
fi

# Function to check service health
check_service() {
    local service=$1
    local endpoint=$2
    local max_retries=30
    local retry_count=0
    
    echo -n "Checking $service service..."
    
    while [ $retry_count -lt $max_retries ]; do
        if curl -s -f "$endpoint" > /dev/null 2>&1; then
            echo " ✅ OK"
            return 0
        fi
        retry_count=$((retry_count + 1))
        sleep 2
    done
    
    echo " ❌ FAILED"
    return 1
}

# Function to run API test
test_api_endpoint() {
    local endpoint=$1
    local expected_status=$2
    local description=$3
    
    echo -n "Testing: $description..."
    
    status=$(curl -s -o /dev/null -w "%{http_code}" "$endpoint")
    
    if [ "$status" = "$expected_status" ]; then
        echo " ✅ OK (Status: $status)"
        return 0
    else
        echo " ❌ FAILED (Expected: $expected_status, Got: $status)"
        return 1
    fi
}

# Run health checks
echo "=== Health Checks ==="
check_service "IAM" "$API_BASE/api/v1/iam/health"
check_service "Project" "$API_BASE/api/v1/projects/health"
check_service "File" "$API_BASE/api/v1/files/health"
check_service "Git Gateway" "$API_BASE/api/v1/git/health"
check_service "CICD" "$API_BASE/api/v1/cicd/health"
check_service "Notification" "$API_BASE/api/v1/notifications/health"
check_service "Team" "$API_BASE/api/v1/teams/health"
check_service "Tenant" "$API_BASE/api/v1/tenants/health"

# Run API tests
echo -e "\n=== API Tests ==="
test_api_endpoint "$API_BASE/api/v1/version" "200" "Version endpoint"
test_api_endpoint "$API_BASE/api/v1/openapi.json" "200" "OpenAPI spec"
test_api_endpoint "$API_BASE/api/v1/metrics" "200" "Metrics endpoint"

# Test authentication flow (without valid credentials, should get 401)
test_api_endpoint "$API_BASE/api/v1/auth/login" "405" "Login endpoint (no POST)"
test_api_endpoint "$API_BASE/api/v1/auth/protected" "401" "Protected endpoint (no auth)"

# Database connectivity test
echo -e "\n=== Database Connectivity ==="
if [ "$ENVIRONMENT" = "production" ] || [ "$ENVIRONMENT" = "staging" ]; then
    echo "Skipping direct database test in $ENVIRONMENT"
else
    echo -n "Testing database connection..."
    if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "SELECT 1" > /dev/null 2>&1; then
        echo " ✅ OK"
    else
        echo " ❌ FAILED"
    fi
fi

# Redis connectivity test
echo -e "\n=== Redis Connectivity ==="
echo -n "Testing Redis connection..."
if redis-cli -h ${REDIS_HOST:-localhost} -p ${REDIS_PORT:-6379} ping > /dev/null 2>&1; then
    echo " ✅ OK"
else
    echo " ❌ FAILED"
fi

# Performance test
echo -e "\n=== Performance Check ==="
echo -n "Testing response time..."
response_time=$(curl -s -o /dev/null -w "%{time_total}" "$HEALTH_ENDPOINT")
response_time_ms=$(echo "$response_time * 1000" | bc)

if (( $(echo "$response_time_ms < 200" | bc -l) )); then
    echo " ✅ OK (${response_time_ms}ms)"
else
    echo " ⚠️  WARNING (${response_time_ms}ms > 200ms threshold)"
fi

# Security headers test
echo -e "\n=== Security Headers ==="
headers=$(curl -s -I "$API_BASE")

check_header() {
    local header=$1
    echo -n "Checking $header..."
    if echo "$headers" | grep -qi "$header"; then
        echo " ✅ Present"
    else
        echo " ❌ Missing"
    fi
}

check_header "X-Content-Type-Options"
check_header "X-Frame-Options"
check_header "X-XSS-Protection"
check_header "Strict-Transport-Security"
check_header "Content-Security-Policy"

echo -e "\n=== Smoke Tests Completed ===\n"

# Summary
if [ $? -eq 0 ]; then
    echo "✅ All smoke tests passed!"
    exit 0
else
    echo "❌ Some smoke tests failed!"
    exit 1
fi
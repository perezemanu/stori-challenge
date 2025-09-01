#!/bin/bash

# Stori Challenge - Lambda Deployment Script
# Usage: ./scripts/deploy-lambda.sh [recipient-email] [function-name]
# Example: ./scripts/deploy-lambda.sh evaluator@example.com stori-processor

set -e

FUNCTION_NAME="${2:-stori-processor}"
RECIPIENT_EMAIL="${1}"
BUILD_DIR="build"
ZIP_FILE="lambda-deployment.zip"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${YELLOW}[STEP]${NC} $1"
}

# Wait for Lambda function to be ready for updates
wait_for_lambda_ready() {
    local function_name="$1"
    local max_wait="${2:-300}"  # Default 5 minutes
    local wait_time=0
    local check_interval=5
    
    log_info "⏳ Checking Lambda function status..."
    
    while [ $wait_time -lt $max_wait ]; do
        local status=$(aws lambda get-function-configuration \
            --function-name "$function_name" \
            --query 'LastUpdateStatus' \
            --output text 2>/dev/null || echo "Unknown")
        
        case "$status" in
            "Successful")
                log_info "✅ Lambda function is ready for updates"
                return 0
                ;;
            "InProgress")
                if [ $wait_time -eq 0 ]; then
                    log_warn "Lambda update in progress, waiting for completion..."
                fi
                echo -n "."
                ;;
            "Failed")
                log_error "Lambda function is in failed state"
                return 1
                ;;
            *)
                log_error "Unknown Lambda status: $status"
                return 1
                ;;
        esac
        
        sleep $check_interval
        wait_time=$((wait_time + check_interval))
    done
    
    echo ""
    log_error "Timeout waiting for Lambda to be ready after ${max_wait}s"
    return 1
}

# Execute Lambda update with retry logic for ResourceConflictException
update_with_retry() {
    local operation="$1"
    local max_retries=5
    local retry_count=0
    local base_delay=10
    
    while [ $retry_count -lt $max_retries ]; do
        if [ $retry_count -gt 0 ]; then
            local delay=$((base_delay * (2 ** (retry_count - 1))))
            log_warn "Retry attempt $retry_count/$max_retries after ${delay}s..."
            sleep $delay
        fi
        
        if ! wait_for_lambda_ready "$FUNCTION_NAME" 60; then
            log_error "Lambda not ready for operation: $operation"
            return 1
        fi
        case "$operation" in
            "code")
                output=$(aws lambda update-function-code \
                    --function-name "$FUNCTION_NAME" \
                    --zip-file "fileb://$ZIP_FILE" 2>&1)
                local exit_code=$?
                ;;
            "config")
                output=$(aws lambda update-function-configuration \
                    --function-name "$FUNCTION_NAME" \
                    --environment "file://$TEMP_ENV_FILE" 2>&1)
                local exit_code=$?
                ;;
            *)
                log_error "Unknown operation: $operation"
                return 1
                ;;
        esac
        
        if [ $exit_code -eq 0 ]; then
            log_info "✅ $operation update completed successfully"
            return 0
        fi
        
        if echo "$output" | grep -q "ResourceConflictException\|update is in progress"; then
            log_warn "⏳ Lambda update conflict detected, will retry..."
            retry_count=$((retry_count + 1))
            continue
        else
            log_error "$operation update failed with non-retryable error"
            return $exit_code
        fi
    done
    
    log_error "Failed to complete $operation update after $max_retries retries"
    return 1
}

if [ -z "$RECIPIENT_EMAIL" ]; then
    log_error "Email recipient is required!"
    echo "Usage: $0 <recipient-email> [function-name]"
    echo "Example: $0 evaluator@example.com stori-processor"
    exit 1
fi
if [[ ! "$RECIPIENT_EMAIL" =~ ^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$ ]]; then
    log_error "Invalid email format: $RECIPIENT_EMAIL"
    exit 1
fi

log_info "Starting Lambda deployment for Stori Challenge"
log_info "Target recipient: $RECIPIENT_EMAIL"
log_info "Lambda function: $FUNCTION_NAME"

if ! command -v aws &> /dev/null; then
    log_error "AWS CLI is not installed. Please install it first."
    exit 1
fi

if ! aws sts get-caller-identity &> /dev/null; then
    log_error "AWS credentials not configured. Please run 'aws configure' first."
    exit 1
fi

log_info "Creating build directory..."
mkdir -p $BUILD_DIR
cd $BUILD_DIR

log_info "Cross-compiling Go binary for Lambda..."
GOOS=linux GOARCH=amd64 go build -o bootstrap ../cmd/lambda/main.go

if [ ! -f "bootstrap" ]; then
    log_error "Failed to compile Lambda binary"
    exit 1
fi

log_info "Creating deployment ZIP..."
rm -f $ZIP_FILE
zip $ZIP_FILE bootstrap

if [ ! -f "$ZIP_FILE" ]; then
    log_error "Failed to create deployment ZIP"
    exit 1
fi

ZIP_SIZE=$(stat -f%z "$ZIP_FILE" 2>/dev/null || stat -c%s "$ZIP_FILE" 2>/dev/null)
log_info "Deployment package size: $(( ZIP_SIZE / 1024 ))KB"

log_step "Step 1: Updating Lambda function code with retry logic..."
if ! update_with_retry "code"; then
    log_error "Failed to update Lambda function code after retries"
    exit 1
fi

if ! wait_for_lambda_ready "$FUNCTION_NAME" 120; then
    log_error "Lambda code update did not complete in time"
    exit 1
fi

log_step "Step 2: Updating Lambda environment variables with retry logic..."
TEMP_ENV_FILE="/tmp/lambda-env-${RANDOM}.json"
cat > "$TEMP_ENV_FILE" << EOF
{
  "Variables": {
    "ENVIRONMENT": "aws",
    "TO_ADDRESS": "$RECIPIENT_EMAIL",
    "SES_FROM_EMAIL": "perezetchegaraymanuel@gmail.com",
    "SES_REGION": "us-west-2"
  }
}
EOF

if ! update_with_retry "config"; then
    rm -f "$TEMP_ENV_FILE"
    log_error "Failed to update Lambda environment variables after retries"
    exit 1
fi

rm -f "$TEMP_ENV_FILE"

if ! wait_for_lambda_ready "$FUNCTION_NAME" 60; then
    log_error "Lambda configuration update did not complete in time"
    exit 1
fi

log_step "Step 3: Verifying environment variables were updated..."
VERIFY_RETRIES=3
VERIFY_COUNT=0
VERIFICATION_SUCCESS=false

while [ $VERIFY_COUNT -lt $VERIFY_RETRIES ]; do
    CURRENT_TO_ADDRESS=$(aws lambda get-function-configuration \
        --function-name "$FUNCTION_NAME" \
        --query 'Environment.Variables.TO_ADDRESS' \
        --output text 2>/dev/null)
    
    if [ "$CURRENT_TO_ADDRESS" = "$RECIPIENT_EMAIL" ]; then
        log_info "✅ Environment variables updated successfully"
        log_info "   TO_ADDRESS: $CURRENT_TO_ADDRESS"
        log_info "   SES_FROM_EMAIL: perezetchegaraymanuel@gmail.com"
        VERIFICATION_SUCCESS=true
        break
    else
        if [ $VERIFY_COUNT -lt $((VERIFY_RETRIES - 1)) ]; then
            log_warn "Configuration not yet propagated, retrying in 5s..."
            sleep 5
        fi
        VERIFY_COUNT=$((VERIFY_COUNT + 1))
    fi
done

if [ "$VERIFICATION_SUCCESS" != "true" ]; then
    log_error "❌ Environment variable update failed or not applied correctly"
    log_error "   Expected: $RECIPIENT_EMAIL"
    log_error "   Current: $CURRENT_TO_ADDRESS"
    log_error "   This may indicate a Lambda service issue or permission problem"
    exit 1
fi

cd ..
rm -rf $BUILD_DIR
log_info "✅ Deployment completed successfully!"
echo ""
echo "📧 Email recipient configured: $RECIPIENT_EMAIL"
echo "🚀 Lambda function: $FUNCTION_NAME"
echo ""
echo "Next steps:"
echo "1. Verify recipient email: ./scripts/verify-emails.sh $RECIPIENT_EMAIL"
echo "2. Test deployment: ./scripts/test-lambda.sh"
echo "3. Upload CSV to trigger: aws s3 cp testdata/transactions.csv s3://your-bucket/input/"
echo ""
log_warn "Note: $RECIPIENT_EMAIL must be verified in AWS SES before receiving emails"
#!/bin/bash

# Stori Challenge - Lambda Deployment Script
# Usage: ./scripts/deploy-lambda.sh [recipient-email] [function-name]
# Example: ./scripts/deploy-lambda.sh evaluator@example.com stori-processor

set -e

# Default values
FUNCTION_NAME="${2:-stori-processor}"
RECIPIENT_EMAIL="${1}"
BUILD_DIR="build"
ZIP_FILE="lambda-deployment.zip"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Validate input
if [ -z "$RECIPIENT_EMAIL" ]; then
    log_error "Email recipient is required!"
    echo "Usage: $0 <recipient-email> [function-name]"
    echo "Example: $0 evaluator@example.com stori-processor"
    exit 1
fi

# Validate email format
if [[ ! "$RECIPIENT_EMAIL" =~ ^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$ ]]; then
    log_error "Invalid email format: $RECIPIENT_EMAIL"
    exit 1
fi

log_info "Starting Lambda deployment for Stori Challenge"
log_info "Target recipient: $RECIPIENT_EMAIL"
log_info "Lambda function: $FUNCTION_NAME"

# Check if AWS CLI is installed
if ! command -v aws &> /dev/null; then
    log_error "AWS CLI is not installed. Please install it first."
    exit 1
fi

# Check AWS credentials
if ! aws sts get-caller-identity &> /dev/null; then
    log_error "AWS credentials not configured. Please run 'aws configure' first."
    exit 1
fi

# Create build directory
log_info "Creating build directory..."
mkdir -p $BUILD_DIR
cd $BUILD_DIR

# Cross-compile for Linux (Lambda runtime)
log_info "Cross-compiling Go binary for Lambda..."
GOOS=linux GOARCH=amd64 go build -o bootstrap ../cmd/lambda/main.go

if [ ! -f "bootstrap" ]; then
    log_error "Failed to compile Lambda binary"
    exit 1
fi

# Create deployment package
log_info "Creating deployment ZIP..."
rm -f $ZIP_FILE
zip $ZIP_FILE bootstrap

# Verify ZIP was created
if [ ! -f "$ZIP_FILE" ]; then
    log_error "Failed to create deployment ZIP"
    exit 1
fi

ZIP_SIZE=$(stat -f%z "$ZIP_FILE" 2>/dev/null || stat -c%s "$ZIP_FILE" 2>/dev/null)
log_info "Deployment package size: $(( ZIP_SIZE / 1024 ))KB"

# Update Lambda function code
log_info "Updating Lambda function code..."
aws lambda update-function-code \
    --function-name $FUNCTION_NAME \
    --zip-file fileb://$ZIP_FILE

if [ $? -ne 0 ]; then
    log_error "Failed to update Lambda function code"
    exit 1
fi

# Update environment variables
log_info "Updating Lambda environment variables..."
aws lambda update-function-configuration \
    --function-name $FUNCTION_NAME \
    --environment Variables="{
        \"ENVIRONMENT\":\"aws\",
        \"TO_ADDRESS\":\"$RECIPIENT_EMAIL\",
        \"SES_FROM_EMAIL\":\"noreply@stori.com\",
        \"SES_REGION\":\"us-west-2\"
    }"

if [ $? -ne 0 ]; then
    log_error "Failed to update Lambda environment variables"
    exit 1
fi

# Cleanup
cd ..
rm -rf $BUILD_DIR

# Success message
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
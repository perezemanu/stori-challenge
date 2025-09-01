#!/bin/bash

# Stori Challenge - Lambda Testing Script
# Usage: ./scripts/test-lambda.sh [function-name] [bucket-name]
# Example: ./scripts/test-lambda.sh stori-processor stori-challenge-bucket

set -e

# Default values
FUNCTION_NAME="${1:-stori-processor}"
BUCKET_NAME="${2:-stori-challenge-manuuu-1756656646}"
TEST_FILE="testdata/transactions.csv"
S3_INPUT_KEY="input/test-transactions-$(date +%s).csv"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

log_info "🧪 Starting Lambda function test"
log_info "Function: $FUNCTION_NAME"
log_info "Bucket: $BUCKET_NAME"
log_info "Test file: $S3_INPUT_KEY"

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

# Check if test CSV file exists
if [ ! -f "$TEST_FILE" ]; then
    log_error "Test CSV file not found: $TEST_FILE"
    log_info "Creating a sample transactions.csv file..."
    
    mkdir -p testdata
    cat > testdata/transactions.csv << EOF
Id,Date,Transaction
1,7/15,+60.5
2,7/28,-10.3
3,8/2,-20.46
4,8/13,+10
EOF
    
    log_info "✅ Sample CSV file created at $TEST_FILE"
fi

# Step 1: Verify Lambda function exists
log_step "Step 1: Verifying Lambda function exists"

aws lambda get-function --function-name $FUNCTION_NAME > /dev/null 2>&1
if [ $? -ne 0 ]; then
    log_error "Lambda function '$FUNCTION_NAME' not found"
    log_info "Available functions:"
    aws lambda list-functions --query 'Functions[].FunctionName' --output table
    exit 1
fi

log_info "✅ Lambda function found: $FUNCTION_NAME"

# Get current configuration
CONFIG=$(aws lambda get-function-configuration --function-name $FUNCTION_NAME)
TIMEOUT=$(echo $CONFIG | jq -r '.Timeout')
MEMORY=$(echo $CONFIG | jq -r '.MemorySize')
RUNTIME=$(echo $CONFIG | jq -r '.Runtime')
TO_EMAIL=$(echo $CONFIG | jq -r '.Environment.Variables.TO_ADDRESS // "Not set"')

log_info "📋 Function Configuration:"
echo "   Runtime: $RUNTIME"
echo "   Memory: ${MEMORY}MB"
echo "   Timeout: ${TIMEOUT}s"
echo "   Email recipient: $TO_EMAIL"

# Step 2: Verify S3 bucket exists
log_step "Step 2: Verifying S3 bucket exists"

aws s3api head-bucket --bucket $BUCKET_NAME > /dev/null 2>&1
if [ $? -ne 0 ]; then
    log_error "S3 bucket '$BUCKET_NAME' not found or not accessible"
    exit 1
fi

log_info "✅ S3 bucket accessible: $BUCKET_NAME"

# Step 3: Upload test file to S3
log_step "Step 3: Uploading test CSV to S3"

aws s3 cp $TEST_FILE s3://$BUCKET_NAME/$S3_INPUT_KEY
if [ $? -ne 0 ]; then
    log_error "Failed to upload test file to S3"
    exit 1
fi

log_info "✅ Test file uploaded: s3://$BUCKET_NAME/$S3_INPUT_KEY"

# Step 4: Monitor Lambda execution
log_step "Step 4: Monitoring Lambda execution"

log_info "⏳ Waiting for Lambda execution (max 30 seconds)..."

# Get log group name
LOG_GROUP="/aws/lambda/$FUNCTION_NAME"

# Wait and monitor logs
WAIT_TIME=0
MAX_WAIT=30

while [ $WAIT_TIME -lt $MAX_WAIT ]; do
    sleep 2
    ((WAIT_TIME += 2))
    
    # Check recent log streams
    RECENT_LOGS=$(aws logs describe-log-streams \
        --log-group-name $LOG_GROUP \
        --order-by LastEventTime \
        --descending \
        --max-items 1 \
        --query 'logStreams[0].logStreamName' \
        --output text 2>/dev/null || echo "")
    
    if [ -n "$RECENT_LOGS" ] && [ "$RECENT_LOGS" != "None" ]; then
        log_info "📋 Recent execution found, checking logs..."
        break
    fi
    
    echo -n "."
done

echo ""

if [ $WAIT_TIME -ge $MAX_WAIT ]; then
    log_warn "No recent Lambda execution detected within ${MAX_WAIT}s"
    log_info "This could be normal if S3 event processing is delayed"
else
    # Get recent logs
    log_info "📄 Recent Lambda logs:"
    aws logs get-log-events \
        --log-group-name $LOG_GROUP \
        --log-stream-name $RECENT_LOGS \
        --limit 20 \
        --query 'events[].[timestamp,message]' \
        --output text | \
        while read timestamp message; do
            if [[ "$timestamp" =~ ^[0-9]+$ ]]; then
                # Convert milliseconds to seconds for date command
                seconds=$((timestamp / 1000))
                formatted_time=$(date -r $seconds '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "Unknown time")
            else
                formatted_time="Unknown time"
            fi
            echo "[$formatted_time] $message"
        done
fi

# Step 5: Check for processed file (optional)
log_step "Step 5: Checking if file was processed"

# Check if file was moved to processed folder (this depends on implementation)
PROCESSED_KEY="processed/$(basename $S3_INPUT_KEY)"
aws s3api head-object --bucket $BUCKET_NAME --key $PROCESSED_KEY > /dev/null 2>&1
if [ $? -eq 0 ]; then
    log_info "✅ File found in processed folder: $PROCESSED_KEY"
else
    log_info "ℹ️  File not found in processed folder (this may be expected)"
fi

# Step 6: Verify email recipient configuration
log_step "Step 6: Email verification status"

if [ "$TO_EMAIL" != "Not set" ] && [ -n "$TO_EMAIL" ]; then
    # Check if email is verified in SES
    SES_STATUS=$(aws ses get-identity-verification-attributes \
        --region us-west-2 \
        --identities $TO_EMAIL \
        --query "VerificationAttributes.\"$TO_EMAIL\".VerificationStatus" \
        --output text 2>/dev/null || echo "NotFound")
    
    case $SES_STATUS in
        "Success")
            log_info "✅ Email recipient verified: $TO_EMAIL"
            ;;
        "Pending")
            log_warn "⏳ Email verification pending: $TO_EMAIL"
            log_info "Recipient needs to check inbox and click verification link"
            ;;
        *)
            log_error "❌ Email not verified: $TO_EMAIL"
            log_info "Run: ./scripts/verify-emails.sh $TO_EMAIL"
            ;;
    esac
else
    log_error "❌ No email recipient configured"
    log_info "Run: ./scripts/deploy-lambda.sh recipient@example.com"
fi

# Summary
echo ""
log_info "🎯 Test Summary:"
echo "   📁 Test file uploaded: s3://$BUCKET_NAME/$S3_INPUT_KEY"
echo "   ⚙️  Lambda function: $FUNCTION_NAME"
echo "   📧 Email recipient: $TO_EMAIL"
echo ""

log_info "📋 Manual verification steps:"
echo "1. Check Lambda CloudWatch Logs for execution details"
echo "2. Verify email delivery to recipient"
echo "3. Check S3 bucket for any processed files"
echo ""

log_info "🔧 Useful commands for debugging:"
echo "   View logs: aws logs tail /aws/lambda/$FUNCTION_NAME --follow"
echo "   List S3 files: aws s3 ls s3://$BUCKET_NAME/ --recursive"
echo "   Check SES stats: aws ses get-send-statistics --region us-west-2"
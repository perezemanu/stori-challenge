#!/bin/bash

# Deployment script for Stori Challenge Lambda
set -e

# Configuration
STACK_NAME="stori-challenge-stack"
REGION="us-west-2"
BUCKET_NAME="stori-challenge-manuuu-1756656646"
SENDER_EMAIL="perezetchegaraymanuel@gmail.com"

echo "🚀 Deploying Stori Challenge to AWS..."
echo "📋 Configuration:"
echo "   Stack Name: $STACK_NAME"
echo "   Region: $REGION"
echo "   Bucket: $BUCKET_NAME"
echo "   Sender Email: $SENDER_EMAIL"
echo ""

# Check if AWS CLI is configured
echo "🔍 Checking AWS configuration..."
if ! aws sts get-caller-identity > /dev/null 2>&1; then
    echo "❌ AWS CLI is not configured or credentials are invalid"
    echo "   Please run: aws configure"
    exit 1
fi

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
echo "✅ AWS configured for account: $ACCOUNT_ID"

# Check if SAM CLI is installed
echo "🔍 Checking SAM CLI..."
if ! command -v sam &> /dev/null; then
    echo "❌ SAM CLI is not installed"
    echo "   Please install: https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/install-sam-cli.html"
    exit 1
fi
echo "✅ SAM CLI found: $(sam --version)"

# Build the project
echo "🏗️  Building project..."
./scripts/build-lambda.sh

# Validate SAM template
echo "🔍 Validating SAM template..."
sam validate --template template.yaml
if [ $? -ne 0 ]; then
    echo "❌ SAM template validation failed"
    exit 1
fi
echo "✅ SAM template is valid"

# Deploy with SAM
echo "🚀 Deploying with SAM..."
sam deploy \
    --template-file template.yaml \
    --stack-name $STACK_NAME \
    --region $REGION \
    --capabilities CAPABILITY_IAM \
    --parameter-overrides \
        SenderEmail=$SENDER_EMAIL \
        BucketName=$BUCKET_NAME \
    --confirm-changeset \
    --resolve-s3

if [ $? -eq 0 ]; then
    echo "✅ Deployment completed successfully!"
    echo ""
    echo "📋 Stack outputs:"
    aws cloudformation describe-stacks \
        --stack-name $STACK_NAME \
        --region $REGION \
        --query 'Stacks[0].Outputs' \
        --output table
    echo ""
    echo "🧪 Test the deployment:"
    echo "   1. Upload a CSV file to: s3://$BUCKET_NAME/input/transactions.csv"
    echo "   2. Check CloudWatch Logs for the Lambda function"
    echo "   3. Verify email delivery to: $SENDER_EMAIL"
else
    echo "❌ Deployment failed"
    exit 1
fi
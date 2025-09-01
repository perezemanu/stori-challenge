#!/bin/bash

# Cleanup script for Stori Challenge Lambda deployment
set -e

STACK_NAME="stori-challenge-stack"
REGION="us-west-2"

echo "🧹 Cleaning up Stori Challenge deployment..."
echo "📋 Configuration:"
echo "   Stack Name: $STACK_NAME"
echo "   Region: $REGION"
echo ""

echo "🔍 Checking AWS configuration..."
if ! aws sts get-caller-identity > /dev/null 2>&1; then
    echo "❌ AWS CLI is not configured or credentials are invalid"
    echo "   Please run: aws configure"
    exit 1
fi

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
echo "✅ AWS configured for account: $ACCOUNT_ID"

echo "🔍 Checking if stack exists..."
if ! aws cloudformation describe-stacks --stack-name $STACK_NAME --region $REGION > /dev/null 2>&1; then
    echo "ℹ️  Stack $STACK_NAME does not exist or already deleted"
    exit 0
fi

# Delete the CloudFormation stack
echo "🗑️  Deleting CloudFormation stack..."
aws cloudformation delete-stack \
    --stack-name $STACK_NAME \
    --region $REGION

echo "⏳ Waiting for stack deletion to complete..."
aws cloudformation wait stack-delete-complete \
    --stack-name $STACK_NAME \
    --region $REGION

if [ $? -eq 0 ]; then
    echo "✅ Stack deleted successfully!"
else
    echo "❌ Failed to delete stack or timeout occurred"
    echo "   Check the CloudFormation console for details"
    exit 1
fi

# Clean up local build artifacts
echo "🧹 Cleaning up local build artifacts..."
rm -rf bin/
rm -f stori-lambda.zip

echo "✅ Cleanup completed!"
echo ""
echo "Note: The S3 bucket and objects are not automatically deleted."
echo "If you want to delete the bucket contents:"
echo "   aws s3 rm s3://stori-challenge-manuuu-1756656646 --recursive"
echo "   aws s3 rb s3://stori-challenge-manuuu-1756656646"
#!/bin/bash

# Stori Challenge - Email Verification Script
# Usage: ./scripts/verify-emails.sh [email1] [email2] [...emailN]
# Example: ./scripts/verify-emails.sh evaluator@example.com reviewer@company.com

set -e

# Configuration
SES_REGION="us-west-2"
FROM_EMAIL="noreply@stori.com"

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

# Check if emails provided
if [ $# -eq 0 ]; then
    log_error "At least one email address is required!"
    echo "Usage: $0 <email1> [email2] [...]"
    echo "Example: $0 evaluator@example.com reviewer@company.com"
    exit 1
fi

# Validate AWS CLI
if ! command -v aws &> /dev/null; then
    log_error "AWS CLI is not installed. Please install it first."
    exit 1
fi

# Check AWS credentials
if ! aws sts get-caller-identity &> /dev/null; then
    log_error "AWS credentials not configured. Please run 'aws configure' first."
    exit 1
fi

log_info "Starting email verification for Stori Challenge"
log_info "SES Region: $SES_REGION"

# First, verify the FROM email (sender)
log_step "Step 1: Verifying sender email ($FROM_EMAIL)"

# Check if sender email is already verified
SENDER_STATUS=$(aws ses get-identity-verification-attributes \
    --region $SES_REGION \
    --identities $FROM_EMAIL \
    --query "VerificationAttributes.\"$FROM_EMAIL\".VerificationStatus" \
    --output text 2>/dev/null || echo "NotFound")

if [ "$SENDER_STATUS" = "Success" ]; then
    log_info "✅ Sender email already verified: $FROM_EMAIL"
elif [ "$SENDER_STATUS" = "Pending" ]; then
    log_warn "⏳ Sender email verification pending: $FROM_EMAIL"
    log_info "Check the inbox for $FROM_EMAIL and click the verification link"
else
    log_info "🔄 Verifying sender email: $FROM_EMAIL"
    aws ses verify-email-identity \
        --region $SES_REGION \
        --email-address $FROM_EMAIL
    
    if [ $? -eq 0 ]; then
        log_info "✅ Verification email sent to: $FROM_EMAIL"
        log_warn "Please check the inbox and click the verification link"
    else
        log_error "Failed to send verification email to: $FROM_EMAIL"
        exit 1
    fi
fi

# Now verify all recipient emails
log_step "Step 2: Verifying recipient email(s)"

VERIFIED_COUNT=0
PENDING_COUNT=0
FAILED_COUNT=0

for EMAIL in "$@"; do
    # Validate email format
    if [[ ! "$EMAIL" =~ ^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$ ]]; then
        log_error "Invalid email format: $EMAIL"
        ((FAILED_COUNT++))
        continue
    fi
    
    # Check current verification status
    STATUS=$(aws ses get-identity-verification-attributes \
        --region $SES_REGION \
        --identities $EMAIL \
        --query "VerificationAttributes.\"$EMAIL\".VerificationStatus" \
        --output text 2>/dev/null || echo "NotFound")
    
    if [ "$STATUS" = "Success" ]; then
        log_info "✅ Already verified: $EMAIL"
        ((VERIFIED_COUNT++))
    elif [ "$STATUS" = "Pending" ]; then
        log_warn "⏳ Verification pending: $EMAIL"
        log_info "Recipient needs to check inbox and click verification link"
        ((PENDING_COUNT++))
    else
        log_info "🔄 Sending verification email to: $EMAIL"
        aws ses verify-email-identity \
            --region $SES_REGION \
            --email-address $EMAIL
        
        if [ $? -eq 0 ]; then
            log_info "📧 Verification email sent to: $EMAIL"
            log_warn "Recipient needs to check inbox and click verification link"
            ((PENDING_COUNT++))
        else
            log_error "Failed to send verification to: $EMAIL"
            ((FAILED_COUNT++))
        fi
    fi
done

# Summary
echo ""
log_info "📊 Verification Summary:"
echo "   ✅ Already verified: $VERIFIED_COUNT"
echo "   ⏳ Pending verification: $PENDING_COUNT"
echo "   ❌ Failed: $FAILED_COUNT"
echo ""

if [ $PENDING_COUNT -gt 0 ]; then
    log_warn "⚠️  Important Notes:"
    echo "   1. Recipients must check their inbox (including spam folder)"
    echo "   2. Verification links expire after 24 hours"
    echo "   3. Re-run this script if verification links expire"
    echo ""
    
    log_info "🔍 To check verification status later, run:"
    for EMAIL in "$@"; do
        echo "   aws ses get-identity-verification-attributes --region $SES_REGION --identities $EMAIL"
    done
fi

# Check SES sending limits
log_step "Step 3: Checking SES sending limits"
SEND_QUOTA=$(aws ses get-send-quota --region $SES_REGION --query 'Max24HourSend' --output text)
SEND_RATE=$(aws ses get-send-quota --region $SES_REGION --query 'MaxSendRate' --output text)
SENT_COUNT=$(aws ses get-send-quota --region $SES_REGION --query 'SentLast24Hours' --output text)

log_info "📈 SES Sending Limits:"
echo "   Daily quota: $SENT_COUNT / $SEND_QUOTA emails"
echo "   Send rate: $SEND_RATE emails/second"

if (( $(echo "$SENT_COUNT >= $SEND_QUOTA" | bc -l) )); then
    log_error "⚠️  Daily sending quota exceeded!"
    echo "   You've sent $SENT_COUNT out of $SEND_QUOTA allowed emails today"
    echo "   Wait 24 hours or request quota increase"
fi

echo ""
if [ $FAILED_COUNT -eq 0 ]; then
    log_info "🎉 Email verification setup completed successfully!"
else
    log_warn "⚠️  Some emails failed verification setup"
    echo "   Review the errors above and try again"
fi
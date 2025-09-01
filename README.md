# Stori Challenge - Transaction Processor

**Financial transaction processor with automated email reports**  
`Go 1.21` • `AWS Lambda` • `SES/Gmail/MailHog` • `Docker`

**Quick Overview**: This README provides a 5-minute quick start guide to evaluate the project.  
 For detailed documentation see [`COMMANDS.md`](./COMMANDS.md)

## 📋 Prerequisites

- **Go 1.21+** and **Docker** installed
- **Make** command available
- **AWS CLI** configured (only for Lambda option)
- **Gmail App Password** (only for Gmail option) - [Create one here](https://support.google.com/accounts/answer/185833)

## 🚀 Quick Start (30 seconds)

### Option 1: MailHog (Easiest - Recommended for evaluators)
```bash
# Process transactions.csv and capture emails for testing
make docker-run-mailhog

# View generated emails at: http://localhost:8025
```

### Option 2: Gmail (Real emails)
```bash
# 1. Copy full configuration
cp .env.example.full .env

# 2. Edit .env - change these lines:
#    EMAIL_MODE=gmail
#    GMAIL_USERNAME=your-email@gmail.com
#    GMAIL_PASSWORD=your-app-password  # NOT your regular password!

# 3. Run with Gmail
make docker-run-gmail
```

### Option 3: AWS Lambda (Production)
```bash
# IMPORTANT: First verify your email with AWS SES
./scripts/verify-emails.sh evaluator@example.com
# Check your inbox and click the AWS verification link

# Then deploy the Lambda function
./scripts/deploy-lambda.sh evaluator@example.com

# (Not a neccesary step, you can ignore) 
#if you want to test it manually and upload the test data to S3
aws s3 cp testdata/transactions.csv s3://your-bucket/input/test.csv
```

---

## 📊 What to Expect

**Input data**: 82 transactions in `testdata/transactions.csv`

**Generated email includes**:
- Total balance: **$17598.95**
- Monthly transactions: January(7), February(7), March(7), etc.
- Average debits: **$-254.09**
- Average credits: **$+1357.94** 
- Professional HTML template with Stori branding

**Where to see results**:
- **MailHog**: http://localhost:8025 (Option 1)
- **Gmail**: Your inbox (Option 2)
- **AWS**: Email configured in deploy script (Option 3)

### 📧 First Email Note
- **First email may take 1-2 minutes** to process and send
- Subsequent emails are instant
- **Check your SPAM folder** if you don't see the email
- Gmail users: Make sure you're using an [App Password](https://support.google.com/accounts/answer/185833), not your regular password

---

## 🔧 Architecture

### Local Development
```
CSV → Go Processor → SQLite → Email (MailHog/Gmail)
```

### AWS Production  
```
S3 Upload → Lambda Trigger → Processor → SES Email
```

**Implemented features**:
- ✅ CSV processing with robust validation
- ✅ Financial calculations with decimal precision
- ✅ Professional responsive HTML email
- ✅ Dual architecture (local + serverless)
- ✅ Complete testing with >90% coverage

---

## 🩺 Quick Troubleshooting

### Can't see emails in MailHog
```bash
# Check containers are running
docker ps

# Restart if needed
make docker-down && make docker-run-mailhog
```

### Gmail not sending
```bash
# Verify configuration
grep -E "EMAIL_MODE|GMAIL_" .env

# Ensure EMAIL_MODE=gmail and you're using an App Password
# Create App Password at: https://support.google.com/accounts/answer/185833
```

### AWS Lambda issues
```bash
# View function logs
aws logs tail /aws/lambda/stori-processor --follow

# Re-deploy if needed (verify email first!)
./scripts/verify-emails.sh your-email@example.com
./scripts/deploy-lambda.sh your-email@example.com
```

### Quick verification
```bash
# Check if processing worked
docker logs stori-challenge-stori-processor-1 | grep "Email sent"

# Should see: "Email sent successfully to: your-email@example.com"
```

---

## 📁 Project Structure

```
stori-challenge/
├── cmd/
│   ├── processor/     # Local/Docker entry point
│   └── lambda/        # AWS Lambda entry point
├── internal/
│   ├── service/       # Business logic (calculator.go)
│   ├── domain/        # Data models
│   └── infrastructure/ # Email, CSV, DB
├── testdata/
│   └── transactions.csv  # Test data
├── scripts/
│   ├── deploy-lambda.sh   # AWS deployment
│   └── verify-emails.sh   # Email verification
└── docker-compose.yml     # Local setup
```

---

## 📋 Additional Commands

```bash
# See all available options
make help

# Run tests
make test

# View test coverage
make test-coverage  # Opens coverage.html

# Clean artifacts
make clean
```

For complete command reference, see `COMMANDS.md`

---

## 🎯 Technical Decisions

**Financial Precision**: `shopspring/decimal` to avoid floating-point errors  
**Professional Email**: HTML templates with complete Stori branding  
**Dual Architecture**: Development flexibility (Docker) + Production scalability (Serverless)  
**Robust Testing**: Unit tests + E2E + data validation

---

**Estimated evaluation time**: 2 minutes setup + 3 minutes review = **5 minutes total**
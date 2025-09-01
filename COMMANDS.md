# 🏗️ Stori Challenge - Commands Reference

Complete command reference for the Stori Transaction Processor project.

## 📦 Make Commands

### Development Workflow
```bash
make deps          # Download and tidy Go dependencies
make fmt           # Format Go code using gofmt
make vet           # Run go vet static analysis
make lint          # Run golangci-lint (if installed)
make test          # Run tests with race detection
make test-coverage # Run tests and generate HTML coverage report
make build         # Build application for Linux (Docker compatible)
make clean         # Clean build artifacts and coverage files
```

### 🐳 Docker - Hybrid Email System

The project uses a **hybrid email system** that allows easy switching between development and production modes:

```bash
# Build Docker image
make docker-build

# Development Mode (Email Testing),
make docker-run-mailhog    # 🧪 MailHog captures emails (no real sending)
                           # Web UI: http://localhost:8025

# Local testing with real email sending via SMPT
make docker-run-gmail      

# Interactive Mode Selection
make docker-run-interactive # 🔄  This allow you to chose between mailhog or gmail(and work same as those above)

# Container Management
make docker-down           # Stop all containers
make docker-clean          # Clean containers and Docker resources
```

### 🛠️ Setup & Help
```bash
make dev-setup    # Setup development environment + golangci-lint
make help         # Show available commands with descriptions
```

---

## 🚀 AWS Lambda Scripts

Located in `./scripts/` directory for AWS deployment:

### Core Lambda Operations
```bash
# Build Lambda function
./scripts/build-lambda.sh
# → Builds Go binary for Linux
# → Creates stori-lambda.zip deployment package
# → Runs tests first

# Deploy to AWS Lambda
./scripts/deploy-lambda.sh recipient@example.com [function-name]
# → Updates Lambda function code and configuration
# → Configures recipient email (TO_ADDRESS)
# → Uses your verified email as sender (perezetchegaraymanuel@gmail.com)
# → Validates configuration was applied correctly

# Test Lambda deployment
./scripts/test-lambda.sh
# → Uploads test CSV to S3
# → Monitors Lambda execution
# → Shows CloudWatch logs
# → Verifies email configuration

# Cleanup AWS resources
./scripts/cleanup.sh
# → Deletes CloudFormation stack
# → Cleans local build artifacts
# → Preserves S3 bucket (manual cleanup)
```

### Additional AWS Scripts
```bash
# Email verification management
./scripts/verify-emails.sh recipient@example.com
# → Initiates SES email verification
# → Checks verification status

```

---

## 🧪 Quick Start Workflows

### Local Development Testing
```bash
# 1. Start with email capture (recommended for development)
make docker-run-mailhog

# 2. View captured emails in browser
open http://localhost:8025

# 3. Process transactions and see results in MailHog UI
```

### Real Email Testing  
```bash
# 1. Switch to real email mode
make docker-run-gmail

# 2. Verify emails are sent to configured recipient
# Check your Gmail inbox for "Stori Account Summary"
```

### AWS Lambda Deployment
```bash
# 1. Build Lambda package
./scripts/build-lambda.sh

# 2. Deploy to AWS with recipient email (sender is automatically your verified email)
./scripts/deploy-lambda.sh recipient@example.com

# 3. Test the deployment
./scripts/test-lambda.sh

# 4. Monitor logs and verify email delivery
```

**Important Notes:**
- **Sender Email**: Automatically uses `perezetchegaraymanuel@gmail.com` (your verified email)
- **Recipient Email**: Must be verified in AWS SES before receiving emails
- **Configuration**: Script validates that email configuration was applied correctly

---

## 📧 Email System Configuration

### Environment Variables
The system uses these key variables (configured in `.env`):

- **`EMAIL_MODE`**: `development` (MailHog) or `gmail` (real emails)
- **`TO_ADDRESS`**: Recipient email address
- **`GMAIL_USERNAME`**: Gmail SMTP username  
- **`GMAIL_PASSWORD`**: Gmail app password
- **`MAILHOG_HOST`**: MailHog server hostname

### Mode Switching
- **Development**: `EMAIL_MODE=development` → Captures emails in MailHog
- **Production**: `EMAIL_MODE=gmail` → Sends real emails via Gmail SMTP

---

## 🔧 Troubleshooting Commands

### Docker Issues
```bash
# View container logs
docker-compose logs -f stori-transaction-processor

# Restart containers
make docker-down && make docker-run-mailhog

# Clean Docker system
make docker-clean
```

### Development Issues
```bash
# Check code quality
make fmt && make vet && make lint

# Run tests with coverage
make test-coverage
open coverage.html

# Clean and rebuild
make clean && make build
```

### AWS Lambda Issues
```bash
# Check Lambda logs
aws logs tail /aws/lambda/stori-processor --follow

# List S3 bucket contents  
aws s3 ls s3://your-bucket-name/ --recursive

# Check SES sending statistics
aws ses get-send-statistics --region us-west-2
```

---

## 💡 Tips

1. **Use MailHog by default** - Avoid spamming real inboxes during development
2. **Check `.env` configuration** - Ensure credentials are properly set
3. **Monitor logs** - Use `docker-compose logs -f` for real-time debugging
4. **Test both modes** - Verify MailHog and Gmail modes work correctly
5. **AWS Prerequisites** - Ensure AWS CLI configured and SES emails verified

---

**📚 Additional Documentation**: See `EMAIL_TESTING_GUIDE.md` for detailed email testing workflows.
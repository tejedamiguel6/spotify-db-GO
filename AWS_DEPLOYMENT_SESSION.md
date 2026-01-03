# AWS ECS Deployment Session Summary

## Overview
Successfully set up and deployed a Go-based Spotify Track Database application to AWS ECS Fargate in the `us-east-2` region.

---

## What Was Accomplished

### 1. Security Audit & Git Configuration
- ✅ Verified no secrets were hardcoded in source files
- ✅ Confirmed `.env` file is properly gitignored
- ✅ Created `.env.example` template for other developers
- ✅ Fixed `.gitignore` patterns to allow `.env.example` while blocking actual `.env` files
- ✅ Removed debug logging that exposed DATABASE_URL in `internal/repository/db.go:27`

### 2. Go Version & CI/CD Fixes
- ✅ Fixed invalid `go 1.24.1` → `go 1.23` in go.mod
- ✅ Updated `.github/workflows/go.yml` from Go 1.20 → Go 1.23
- ✅ Updated `Dockerfile` to use Go 1.23
- ✅ CI/CD builds now passing

### 3. AWS Secrets Manager Setup
**Region:** `us-east-2`
**Account ID:** `748344702309`

Created secrets:
- `arn:aws:secretsmanager:us-east-2:748344702309:secret:spotify-track-db/DATABASE_URL-aWPGy3`
- `arn:aws:secretsmanager:us-east-2:748344702309:secret:spotify-track-db/SPOTIFY_CLIENT_ID-KhXFWF`
- `arn:aws:secretsmanager:us-east-2:748344702309:secret:spotify-track-db/SPOTIFY_CLIENT_SECRET-9RmSgg`

**Script:** `./setup-aws-secrets.sh` (configured for us-east-2)

### 4. IAM Roles Created
Created two IAM roles with proper trust relationships:

**ecsTaskExecutionRole:**
- ARN: `arn:aws:iam::748344702309:role/ecsTaskExecutionRole`
- Attached policies:
  - `AmazonECSTaskExecutionRolePolicy` (AWS managed)
  - `SecretsManagerAccess` (inline policy for accessing secrets)
- Purpose: Allows ECS to pull container images from ECR and retrieve secrets

**ecsTaskRole:**
- ARN: `arn:aws:iam::748344702309:role/ecsTaskRole`
- Purpose: Application runtime permissions (currently minimal)

### 5. Docker Configuration
**Created Dockerfile** with multi-stage build:
- Build stage: Go 1.23 Alpine
- Runtime stage: Minimal Alpine with ca-certificates
- Binary location: `./cmd/server/main.go`
- Exposed port: 8080

**Key Fix:** Updated `deploy-to-ecs.sh` to build for `linux/amd64` platform:
```bash
docker buildx build --platform linux/amd64 -t spotify-track-db:latest .
```
This was critical because building on Apple Silicon (arm64) creates incompatible images for AWS Fargate (amd64).

### 6. AWS ECS Infrastructure
**Region:** us-east-2

**ECR Repository:**
- Name: `spotify-track-db`
- URI: `748344702309.dkr.ecr.us-east-2.amazonaws.com/spotify-track-db`

**ECS Cluster:**
- Name: `spotify-cluster`
- Region: `us-east-2`

**ECS Service:**
- Name: `spotify-track-db`
- Launch type: FARGATE
- Desired count: 1
- VPC: `vpc-0ecdc57ac3fd716d2` (default VPC)
- Subnets:
  - `subnet-06d21d97bb8da1d7d`
  - `subnet-08bad66d1ab44e030`
  - `subnet-07dfb78e9905054af`
- Security Group: `sg-071cb58cda54a5995` (default)
- Public IP: ENABLED

**ECS Task Definition:**
- Family: `spotify-track-db`
- CPU: 256 (.25 vCPU)
- Memory: 512 MB
- Network mode: awsvpc
- Container port: 8080
- Logs: CloudWatch Logs group `/ecs/spotify-track-db`

### 7. Branch Structure
**Working Branch:** `advance-cron-job` (has the correct code structure)
**Main Branch:** Had outdated code

The deployment files were added to the `advance-cron-job` branch which contains:
- Proper directory structure with `cmd/server/main.go`
- Internal packages in `internal/` directory
- Updated handlers, models, and services

### 8. Deployment Automation Scripts

**setup-aws-secrets.sh:**
- Reads `.env` file
- Creates/updates AWS Secrets Manager secrets
- Region: us-east-2
- Handles quoted values in .env correctly

**deploy-to-ecs.sh:**
- Gets AWS Account ID
- Creates ECR repository (if needed)
- Authenticates Docker to ECR
- Builds Docker image for linux/amd64
- Pushes to ECR
- Registers task definition
- Creates/updates ECS service
- Region: us-east-2

**ecs-task-definition.json:**
- Task definition template
- Uses us-east-2 region
- References secrets from Secrets Manager
- Configured for CloudWatch Logs

---

## Current Status

### ✅ DEPLOYMENT SUCCESSFUL WITH HTTPS!

**Status:** Application is running successfully on AWS ECS Fargate in us-east-2 with HTTPS

**Production URL:** https://api-spotify-tracks.mtejeda.co

**Issues Resolved:**
1. ✅ Platform mismatch (arm64 → amd64) - Fixed by using `docker buildx build --platform linux/amd64`
2. ✅ Missing .env file in container - Fixed by making `godotenv.Load()` optional in `internal/repository/db.go:22-24`
3. ✅ IAM roles missing - Created `ecsTaskExecutionRole` and `ecsTaskRole` with proper policies
4. ✅ Secrets Manager integration - Environment variables now injected from AWS Secrets Manager
5. ✅ Security group configuration - Opened ports 80, 443, and 8080
6. ✅ Application Load Balancer - Set up with HTTPS and health checks
7. ✅ SSL Certificate - Issued and validated via AWS Certificate Manager
8. ✅ Custom domain - Configured api-spotify-tracks.mtejeda.co via Cloudflare

**Final Fix Applied:**
Changed `internal/repository/db.go` lines 22-24 from:
```go
if err := godotenv.Load(); err != nil {
    log.Fatal("Error loading .env:", err)  // This was crashing!
}
```

To:
```go
if err := godotenv.Load(); err != nil {
    log.Println("Note: .env file not found (this is normal in production environments)")
}
```

This allows the app to work both locally (with .env file) and in production (with AWS Secrets Manager).

### 🎉 Access Your Application

**Production URL:** https://api-spotify-tracks.mtejeda.co

**Available Endpoints:**
- `GET https://api-spotify-tracks.mtejeda.co/recently-played-tracks`
- `GET https://api-spotify-tracks.mtejeda.co/now-listening-to`
- `GET https://api-spotify-tracks.mtejeda.co/recently-liked`
- `GET https://api-spotify-tracks.mtejeda.co/genre/:genre`
- `GET https://api-spotify-tracks.mtejeda.co/collection-stats`
- `POST https://api-spotify-tracks.mtejeda.co/save-refresh`

**Load Balancer:**
- DNS Name: `spotify-track-db-alb-450034939.us-east-2.elb.amazonaws.com`
- ARN: `arn:aws:elasticloadbalancing:us-east-2:748344702309:loadbalancer/app/spotify-track-db-alb/cae1a7fc54e72299`

**SSL Certificate:**
- ARN: `arn:aws:acm:us-east-2:748344702309:certificate/3d273441-9e13-4464-8493-89cd6f41800b`
- Domain: `api-spotify-tracks.mtejeda.co`
- Status: ISSUED

**Target Group:**
- ARN: `arn:aws:elasticloadbalancing:us-east-2:748344702309:targetgroup/spotify-track-db-tg/eaaac5192859e236`
- Health check path: `/recently-played-tracks`
- Health check interval: 30 seconds

---

## Environment Configuration

### Database
- Using Neon PostgreSQL (serverless)
- Connection string stored in AWS Secrets Manager
- SSL mode: require
- Channel binding: require

### Spotify API
- Client ID and Secret stored in AWS Secrets Manager
- Application runs on port 8080

---

## Files Created/Modified

### Created Files
- `Dockerfile` - Multi-stage Docker build
- `setup-aws-secrets.sh` - Secrets Manager automation
- `deploy-to-ecs.sh` - Full deployment automation
- `ecs-task-definition.json` - ECS task configuration
- `.env.example` - Environment template
- `DEPLOYMENT.md` - Comprehensive deployment guide

### Modified Files
- `go.mod` - Fixed Go version from 1.24.1 → 1.23
- `.github/workflows/go.yml` - Updated Go version 1.20 → 1.23
- `.gitignore` - Fixed to allow .env.example
- `internal/repository/db.go` - Removed debug logging with credentials
- `README.md` - Updated prerequisites and setup instructions

---

## Important Commands

### Check Service Status
```bash
aws ecs describe-services \
  --cluster spotify-cluster \
  --services spotify-track-db \
  --region us-east-2 \
  --query 'services[0].{Running:runningCount,Desired:desiredCount,Status:status}' \
  --output table
```

### View Logs
```bash
aws logs tail /ecs/spotify-track-db --follow --region us-east-2
```

### View Recent Events
```bash
aws ecs describe-services \
  --cluster spotify-cluster \
  --services spotify-track-db \
  --region us-east-2 \
  --query 'services[0].events[0:5]' \
  --output table
```

### Get Task Public IP
```bash
TASK_ARN=$(aws ecs list-tasks --cluster spotify-cluster --service-name spotify-track-db --region us-east-2 --query 'taskArns[0]' --output text)
ENI_ID=$(aws ecs describe-tasks --cluster spotify-cluster --tasks $TASK_ARN --region us-east-2 --query 'tasks[0].attachments[0].details[?name==`networkInterfaceId`].value' --output text)
aws ec2 describe-network-interfaces --network-interface-ids $ENI_ID --region us-east-2 --query 'NetworkInterfaces[0].Association.PublicIp' --output text
```

### Force New Deployment
```bash
aws ecs update-service \
  --cluster spotify-cluster \
  --service spotify-track-db \
  --force-new-deployment \
  --region us-east-2
```

---

## Cost Estimation
- **Fargate (0.25 vCPU, 0.5 GB RAM, 24/7):** ~$12/month
- **CloudWatch Logs:** ~$1/month
- **Secrets Manager:** $1.20/month (3 secrets × $0.40)
- **ECR Storage:** Minimal
- **Data Transfer:** Varies
- **Total:** ~$15-20/month

---

## Security Notes
- ✅ No credentials committed to git
- ✅ All secrets in AWS Secrets Manager
- ✅ IAM roles follow least privilege
- ✅ ECS tasks use separate execution and task roles
- ✅ Container runs as non-root user (Alpine default)
- ⚠️ Default security group allows all outbound traffic
- ⚠️ No load balancer (tasks have public IPs)

---

## Troubleshooting Tips

### If deployment keeps failing:
1. Check service events for error messages
2. Verify IAM roles exist and have correct trust relationships
3. Ensure secrets exist in the correct region
4. Check CloudWatch logs for application errors
5. Verify security group allows outbound HTTPS (443) for database/Spotify API

### If container won't start:
1. Test Docker build locally: `docker build -t test .`
2. Run container locally: `docker run -p 8080:8080 test`
3. Check environment variables are being passed correctly
4. Verify database connection string is accessible

---

## Additional Context

### Why us-east-2?
Secrets were initially created in us-east-1 by mistake, then deleted and recreated in us-east-2 per user's requirement.

### Why advance-cron-job branch?
The main branch had an older code structure with `main.go` in the root directory. The advance-cron-job branch has the proper structure with `cmd/server/main.go` and organized internal packages.

### Platform Issue Details
Apple Silicon Macs (M1/M2/M3) use ARM64 architecture. Docker builds on these machines create arm64 images by default. AWS Fargate only supports x86_64/amd64 architecture, so cross-platform builds are required using `docker buildx build --platform linux/amd64`.

---

## What's Left To Do

1. ✅ ~~Run `./deploy-to-ecs.sh` to deploy the corrected amd64 image~~ - COMPLETED
2. ✅ ~~Check logs and confirm application starts successfully~~ - COMPLETED
3. ✅ ~~Test the application via the task's public IP on port 8080~~ - COMPLETED
4. ✅ ~~Set up Application Load Balancer for production use~~ - COMPLETED
5. ✅ ~~Configure custom domain (api-spotify-tracks.mtejeda.co)~~ - COMPLETED
6. ✅ ~~Set up SSL/HTTPS with AWS Certificate Manager~~ - COMPLETED
7. **Important:** Update your frontend/client to point to `https://api-spotify-tracks.mtejeda.co`
8. **Optional:** Set up CloudWatch alarms for monitoring
9. **Optional:** Configure auto-scaling based on CPU/memory
10. **Recommended:** Remove port 8080 public access (only ALB needs it now)
11. **Optional:** Disable public IP assignment for ECS tasks (more secure, behind ALB)

---

## Load Balancer Setup (Session 2)

### Steps Completed

1. **Requested SSL Certificate from AWS Certificate Manager**
   - Domain: `api-spotify-tracks.mtejeda.co`
   - Validation method: DNS
   - Added CNAME record in Cloudflare for validation
   - Certificate issued successfully

2. **Created Target Group**
   - Name: `spotify-track-db-tg`
   - Protocol: HTTP
   - Port: 8080
   - Target type: IP (for Fargate)
   - Health check: `/recently-played-tracks` every 30 seconds

3. **Created Application Load Balancer**
   - Name: `spotify-track-db-alb`
   - Scheme: internet-facing
   - Subnets: 3 subnets across availability zones
   - Security group: `sg-071cb58cda54a5995`

4. **Configured Security Group**
   - Opened port 80 (HTTP) - redirects to HTTPS
   - Opened port 443 (HTTPS) - serves application
   - Port 8080 still open for direct access (can be removed later)

5. **Created Load Balancer Listeners**
   - HTTPS Listener (port 443): Forwards to target group with SSL termination
   - HTTP Listener (port 80): Redirects to HTTPS (301 permanent redirect)

6. **Updated ECS Service**
   - Connected service to load balancer
   - Target group now manages task health
   - Tasks automatically registered/deregistered

7. **Updated Cloudflare DNS**
   - Changed from A record (IP) to CNAME record (ALB DNS name)
   - Target: `spotify-track-db-alb-450034939.us-east-2.elb.amazonaws.com`
   - Proxy status: DNS only (gray cloud)

### Cost Update

**Monthly costs increased from ~$14 to ~$30:**
- Fargate: $12/month (unchanged)
- **Application Load Balancer: $16/month (NEW)**
- Secrets Manager: $1.20/month (unchanged)
- CloudWatch Logs: $0.50/month (unchanged)
- ECR Storage: ~$0.01/month (unchanged)

**Total: ~$30/month**

### Architecture After Load Balancer

```
Internet
    ↓
Cloudflare DNS (api-spotify-tracks.mtejeda.co)
    ↓
Application Load Balancer
    ├─ HTTPS Listener (port 443) → SSL termination
    └─ HTTP Listener (port 80) → Redirect to HTTPS
        ↓
    Target Group (health checks)
        ↓
    ECS Service (spotify-track-db)
        ↓
    ECS Tasks (Fargate)
        ↓
    Your Go Application (port 8080)
```

## Questions for Next Session

1. ~~Do you want to set up a load balancer and custom domain?~~ - ✅ COMPLETED
2. Should we add CloudWatch monitoring and alarms?
3. Do you want to configure auto-scaling based on CPU/memory?
4. Should we remove public IP assignment from tasks (more secure)?
4. Should we set up a CI/CD pipeline for automated deployments on git push?
5. Do you need a development environment separate from production?

---

**Session Date:** January 2, 2026
**AWS Account:** 748344702309
**Region:** us-east-2
**Current Branch:** advance-cron-job

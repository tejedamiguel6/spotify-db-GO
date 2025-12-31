#!/bin/bash

# Setup automated backup scheduling for Spotify Track Database
# This will schedule backups to run every 3 weeks

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}⏰ Setting up automated backup scheduling${NC}"

# Get the absolute path to the backup script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BACKUP_SCRIPT="$SCRIPT_DIR/backup.sh"

if [ ! -f "$BACKUP_SCRIPT" ]; then
    echo -e "${RED}❌ Backup script not found at: $BACKUP_SCRIPT${NC}"
    exit 1
fi

# Make sure backup script is executable
chmod +x "$BACKUP_SCRIPT"

echo -e "${GREEN}✅ Found backup script at: $BACKUP_SCRIPT${NC}"
echo -e "${GREEN}✅ Project directory: $PROJECT_DIR${NC}"

# Create a wrapper script that changes to the project directory
WRAPPER_SCRIPT="$SCRIPT_DIR/backup-wrapper.sh"
cat > "$WRAPPER_SCRIPT" << EOF
#!/bin/bash
# Backup wrapper script - changes to project directory before running backup

cd "$PROJECT_DIR" || exit 1
exec "$BACKUP_SCRIPT"
EOF

chmod +x "$WRAPPER_SCRIPT"
echo -e "${GREEN}✅ Created wrapper script: $WRAPPER_SCRIPT${NC}"

# Determine the cron schedule
# Every 3 weeks = every 21 days
# We'll schedule it for Sundays at 2 AM to avoid disrupting usage
CRON_SCHEDULE="0 2 */21 * *"

# Create the cron job entry
CRON_JOB="$CRON_SCHEDULE $WRAPPER_SCRIPT >> $PROJECT_DIR/logs/backup.log 2>&1"

echo -e "${YELLOW}📋 Proposed cron job:${NC}"
echo -e "${BLUE}$CRON_JOB${NC}"
echo -e "${YELLOW}📅 This will run every 3 weeks on Sunday at 2:00 AM${NC}"

# Create logs directory
mkdir -p "$PROJECT_DIR/logs"

# Check if cron job already exists
if crontab -l 2>/dev/null | grep -q "$WRAPPER_SCRIPT"; then
    echo -e "${YELLOW}⚠️  Existing backup cron job found${NC}"
    echo -e "${YELLOW}🔄 Updating existing cron job...${NC}"
    # Remove existing job and add new one
    (crontab -l 2>/dev/null | grep -v "$WRAPPER_SCRIPT"; echo "$CRON_JOB") | crontab -
else
    echo -e "${YELLOW}➕ Adding new cron job...${NC}"
    # Add new job to existing crontab
    (crontab -l 2>/dev/null; echo "$CRON_JOB") | crontab -
fi

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Cron job scheduled successfully!${NC}"
else
    echo -e "${RED}❌ Failed to schedule cron job${NC}"
    exit 1
fi

# Show current crontab
echo -e "${BLUE}📋 Current cron jobs:${NC}"
crontab -l | grep -E "(backup|spotify)" || echo "  (No backup-related cron jobs found)"

# Test the backup script
echo -e "${YELLOW}🧪 Testing backup script...${NC}"
echo -e "${YELLOW}Would you like to run a test backup now? (y/n)${NC}"
read -r response

if [[ "$response" =~ ^[Yy]$ ]]; then
    echo -e "${BLUE}🔄 Running test backup...${NC}"
    cd "$PROJECT_DIR"
    "$BACKUP_SCRIPT"
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ Test backup completed successfully!${NC}"
    else
        echo -e "${RED}❌ Test backup failed${NC}"
    fi
else
    echo -e "${YELLOW}⏩ Skipping test backup${NC}"
fi

echo -e "${GREEN}🎉 Backup scheduling setup complete!${NC}"
echo -e "${BLUE}📋 Summary:${NC}"
echo -e "${BLUE}  • Backups will run every 3 weeks (21 days)${NC}"
echo -e "${BLUE}  • Scheduled for Sundays at 2:00 AM${NC}"  
echo -e "${BLUE}  • Backups stored in: $HOME/spotify-db-backups${NC}"
echo -e "${BLUE}  • Keeps 4 most recent backups (≈3 months)${NC}"
echo -e "${BLUE}  • Logs stored in: $PROJECT_DIR/logs/backup.log${NC}"

echo -e "${YELLOW}💡 Useful commands:${NC}"
echo -e "${YELLOW}  • View cron jobs: crontab -l${NC}"
echo -e "${YELLOW}  • Remove backup job: crontab -e (then delete the line)${NC}"
echo -e "${YELLOW}  • Run backup manually: cd $PROJECT_DIR && ./scripts/backup.sh${NC}"
echo -e "${YELLOW}  • View backup logs: tail -f $PROJECT_DIR/logs/backup.log${NC}"
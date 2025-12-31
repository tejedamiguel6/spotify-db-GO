#!/bin/bash

# Spotify Track Database Backup Script
# Runs every 3 weeks to backup your precious listening data

set -e  # Exit on any error

# Configuration
BACKUP_DIR="$HOME/spotify-db-backups"
DATE=$(date +%Y-%m-%d_%H-%M-%S)
BACKUP_NAME="spotify-tracks-backup_$DATE"
KEEP_BACKUPS=4  # Keep 4 backups (about 3 months worth)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🗄️  Starting Spotify Track Database Backup${NC}"
echo -e "${BLUE}📅 Date: $(date)${NC}"
echo -e "${BLUE}📁 Backup directory: $BACKUP_DIR${NC}"

# Load environment variables
if [ -f .env ]; then
    source .env
    echo -e "${GREEN}✅ Loaded .env file${NC}"
else
    echo -e "${RED}❌ .env file not found. Make sure you're in the project directory.${NC}"
    exit 1
fi

# Check if DATABASE_URL exists
if [ -z "$DATABASE_URL" ]; then
    echo -e "${RED}❌ DATABASE_URL not found in .env file${NC}"
    exit 1
fi

# Create backup directory if it doesn't exist
mkdir -p "$BACKUP_DIR"

# Create full backup directory for this backup
FULL_BACKUP_PATH="$BACKUP_DIR/$BACKUP_NAME"
mkdir -p "$FULL_BACKUP_PATH"

echo -e "${YELLOW}🔄 Creating database dump...${NC}"

# Create PostgreSQL dump
pg_dump "$DATABASE_URL" > "$FULL_BACKUP_PATH/database.sql"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Database dump created successfully${NC}"
else
    echo -e "${RED}❌ Database dump failed${NC}"
    exit 1
fi

# Create backup info file
cat > "$FULL_BACKUP_PATH/backup_info.txt" << EOF
Spotify Track Database Backup
=============================

Backup Date: $(date)
Backup Name: $BACKUP_NAME
Database URL: ${DATABASE_URL%@*}@[HIDDEN]

Tables backed up:
- recently_played (listening history)
- recently_liked (saved tracks)
- spotify_auth (authentication tokens)
- tracks_on_repeat (legacy data)

Total backup size: $(du -sh "$FULL_BACKUP_PATH" | cut -f1)

To restore this backup:
1. Make sure PostgreSQL is running
2. Create a new database or drop existing data
3. Run: psql "\$DATABASE_URL" < database.sql

Created by: Spotify Track DB Backup Script v1.0
EOF

# Get backup statistics
echo -e "${YELLOW}📊 Gathering backup statistics...${NC}"

# Count records in main tables
RECENTLY_PLAYED_COUNT=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM recently_played;" | xargs)
RECENTLY_LIKED_COUNT=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM recently_liked;" | xargs)

# Add statistics to info file
cat >> "$FULL_BACKUP_PATH/backup_info.txt" << EOF

Backup Statistics:
==================
Recently Played Tracks: $RECENTLY_PLAYED_COUNT
Recently Liked Tracks:  $RECENTLY_LIKED_COUNT

Latest Track Dates:
EOF

# Get latest dates
LATEST_PLAYED=$(psql "$DATABASE_URL" -t -c "SELECT MAX(played_at) FROM recently_played;" | xargs)
LATEST_LIKED=$(psql "$DATABASE_URL" -t -c "SELECT MAX(added_at) FROM recently_liked;" | xargs)

cat >> "$FULL_BACKUP_PATH/backup_info.txt" << EOF
Latest Recently Played: $LATEST_PLAYED
Latest Recently Liked:  $LATEST_LIKED
EOF

# Compress the backup
echo -e "${YELLOW}🗜️  Compressing backup...${NC}"
cd "$BACKUP_DIR"
tar -czf "${BACKUP_NAME}.tar.gz" "$BACKUP_NAME"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Backup compressed successfully${NC}"
    # Remove uncompressed directory
    rm -rf "$BACKUP_NAME"
else
    echo -e "${RED}❌ Backup compression failed${NC}"
    exit 1
fi

# Calculate final backup size
BACKUP_SIZE=$(du -sh "${BACKUP_NAME}.tar.gz" | cut -f1)

echo -e "${GREEN}🎉 Backup completed successfully!${NC}"
echo -e "${GREEN}📁 Backup location: $BACKUP_DIR/${BACKUP_NAME}.tar.gz${NC}"
echo -e "${GREEN}📏 Backup size: $BACKUP_SIZE${NC}"
echo -e "${GREEN}🎵 Recently Played Tracks: $RECENTLY_PLAYED_COUNT${NC}"
echo -e "${GREEN}💚 Recently Liked Tracks: $RECENTLY_LIKED_COUNT${NC}"

# Clean up old backups (keep only the most recent ones)
echo -e "${YELLOW}🧹 Cleaning up old backups...${NC}"

cd "$BACKUP_DIR"
BACKUP_COUNT=$(ls -1 spotify-tracks-backup_*.tar.gz 2>/dev/null | wc -l)

if [ $BACKUP_COUNT -gt $KEEP_BACKUPS ]; then
    echo -e "${YELLOW}📦 Found $BACKUP_COUNT backups, keeping newest $KEEP_BACKUPS${NC}"
    ls -t spotify-tracks-backup_*.tar.gz | tail -n +$(($KEEP_BACKUPS + 1)) | xargs rm -f
    echo -e "${GREEN}✅ Old backups cleaned up${NC}"
else
    echo -e "${GREEN}✅ Only $BACKUP_COUNT backups found, no cleanup needed${NC}"
fi

# List current backups
echo -e "${BLUE}📋 Current backups:${NC}"
ls -lah spotify-tracks-backup_*.tar.gz | awk '{print "  " $9 " (" $5 ", " $6 " " $7 ")"}'

echo -e "${GREEN}🚀 Backup process completed successfully!${NC}"
echo -e "${BLUE}💡 To restore: tar -xzf ${BACKUP_NAME}.tar.gz && psql \$DATABASE_URL < ${BACKUP_NAME}/database.sql${NC}"
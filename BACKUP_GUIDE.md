# 🗄️ Spotify Track Database Backup Guide

Never lose your precious listening data again! This guide covers automated backups, manual backups, and disaster recovery.

## 🚀 Quick Setup (Automated Backups Every 3 Weeks)

```bash
# 1. Run the setup script
./scripts/setup-backup-schedule.sh

# 2. Follow the prompts to schedule automatic backups
```

**That's it!** Your database will now be backed up automatically every 3 weeks.

---

## 📋 What Gets Backed Up

✅ **recently_played** - Your complete listening history  
✅ **recently_liked** - All your saved/liked tracks with genres  
✅ **spotify_auth** - Authentication tokens  
✅ **tracks_on_repeat** - Legacy track data  
✅ **Database schema** - Table structures and indexes  

## 📅 Backup Schedule

- **Frequency**: Every 3 weeks (21 days)
- **Time**: Sundays at 2:00 AM  
- **Location**: `~/spotify-db-backups/`
- **Retention**: Keeps 4 most recent backups (~3 months)
- **Format**: Compressed `.tar.gz` files

---

## 🔧 Manual Backup

### Run a backup right now:
```bash
cd /path/to/your/spotify-track-db
./scripts/backup.sh
```

### What happens:
1. 🗄️ Creates PostgreSQL dump using `pg_dump`
2. 📊 Collects database statistics (track counts, dates)
3. 📝 Creates detailed backup info file
4. 🗜️ Compresses everything into `.tar.gz`
5. 🧹 Cleans up old backups (keeps newest 4)

---

## 🔄 Restore From Backup

### 1. List Available Backups
```bash
ls -la ~/spotify-db-backups/
```

### 2. Extract Backup
```bash
cd ~/spotify-db-backups
tar -xzf spotify-tracks-backup_2024-08-22_14-30-15.tar.gz
```

### 3. Restore Database
```bash
# Option A: Restore to existing database (DESTRUCTIVE)
psql $DATABASE_URL < spotify-tracks-backup_2024-08-22_14-30-15/database.sql

# Option B: Create new database first (SAFER)
createdb spotify_tracks_restored
psql postgresql://user:pass@host/spotify_tracks_restored < spotify-tracks-backup_2024-08-22_14-30-15/database.sql
```

### 4. Update .env (if using new database)
```bash
# Update DATABASE_URL in .env file to point to restored database
DATABASE_URL=postgresql://user:pass@host/spotify_tracks_restored
```

---

## 📊 Backup Contents

Each backup contains:

### `database.sql`
- Complete PostgreSQL database dump
- All tables, data, indexes, and constraints
- Ready to restore with `psql`

### `backup_info.txt`
- Backup metadata (date, size, table counts)
- Latest track dates for verification
- Restore instructions
- Statistics summary

Example backup info:
```
Spotify Track Database Backup
=============================

Backup Date: Wed Aug 22 14:30:15 PDT 2024
Recently Played Tracks: 2,847
Recently Liked Tracks: 1,205
Latest Recently Played: 2024-08-22 14:25:33
Latest Recently Liked: 2024-08-22 10:15:22
Total backup size: 4.2M
```

---

## 🛠️ Backup Management

### View Current Schedule
```bash
crontab -l | grep backup
```

### Manually Run Test Backup
```bash
cd /path/to/spotify-track-db
./scripts/backup.sh
```

### View Backup Logs
```bash
tail -f logs/backup.log
```

### Remove Automated Backups
```bash
crontab -e
# Delete the line containing backup-wrapper.sh
```

### Change Backup Frequency
Edit the cron schedule in `setup-backup-schedule.sh`:
```bash
# Every 2 weeks instead of 3
CRON_SCHEDULE="0 2 */14 * *"

# Every week
CRON_SCHEDULE="0 2 * * 0"  # Every Sunday

# Every day at 3 AM
CRON_SCHEDULE="0 3 * * *"
```

---

## 🚨 Disaster Recovery Scenarios

### Scenario 1: Database Corruption
```bash
# 1. Stop your application
pkill -f "spotify-track-db"

# 2. Restore from latest backup
cd ~/spotify-db-backups
tar -xzf $(ls -t spotify-tracks-backup_*.tar.gz | head -1)
BACKUP_DIR=$(ls -td spotify-tracks-backup_* | head -1)
psql $DATABASE_URL < $BACKUP_DIR/database.sql

# 3. Restart application
./bin/server
```

### Scenario 2: Accidental Data Deletion
```bash
# Restore specific tables only
psql $DATABASE_URL << EOF
DROP TABLE recently_played;
DROP TABLE recently_liked;
EOF

# Extract and restore just the data you need
psql $DATABASE_URL < backup/database.sql
```

### Scenario 3: Database Server Migration
```bash
# 1. Create backup on old server
./scripts/backup.sh

# 2. Copy backup to new server
scp ~/spotify-db-backups/latest_backup.tar.gz new-server:~/

# 3. Restore on new server
tar -xzf latest_backup.tar.gz
createdb spotify_tracks
psql postgresql://user:pass@new-server/spotify_tracks < database.sql

# 4. Update .env with new DATABASE_URL
```

---

## 🔍 Verifying Backups

### Check Backup Integrity
```bash
# Test that backup can be extracted
cd ~/spotify-db-backups
tar -tzf spotify-tracks-backup_2024-08-22_14-30-15.tar.gz

# Verify SQL file syntax
pg_dump --schema-only $DATABASE_URL | psql --set ON_ERROR_STOP=on -f - template1 >/dev/null
```

### Compare Backup with Current Data
```bash
# Check if backup has expected number of records
grep "Recently Played Tracks:" ~/spotify-db-backups/*/backup_info.txt | tail -1
psql $DATABASE_URL -c "SELECT COUNT(*) FROM recently_played;"
```

---

## ⚡ Pro Tips

1. **Test Your Backups**: Regularly verify you can restore from backups
2. **Monitor Disk Space**: Backups are compressed but still take space
3. **Off-site Storage**: Consider copying backups to cloud storage
4. **Multiple Schedules**: Run daily backups during active development
5. **Pre-Migration Backups**: Always backup before major changes

---

## 🆘 Troubleshooting

### "Command not found: pg_dump"
```bash
# macOS
brew install postgresql

# Ubuntu/Debian  
sudo apt install postgresql-client

# CentOS/RHEL
sudo yum install postgresql
```

### "Permission denied"
```bash
chmod +x scripts/*.sh
```

### "Cron job not running"
```bash
# Check cron service is running
sudo service cron status

# Check cron logs
tail -f /var/log/cron

# Test cron job manually
/path/to/your/scripts/backup-wrapper.sh
```

### "Database connection failed"
```bash
# Verify DATABASE_URL in .env
echo $DATABASE_URL

# Test connection manually
psql $DATABASE_URL -c "SELECT 1;"
```

---

## 📞 Quick Reference

| Command | Purpose |
|---------|---------|
| `./scripts/backup.sh` | Manual backup now |
| `./scripts/setup-backup-schedule.sh` | Setup automated backups |
| `crontab -l` | View scheduled backups |
| `ls ~/spotify-db-backups/` | List all backups |
| `tail -f logs/backup.log` | Watch backup logs |

**Your data is now safe! 🎉** The backup system will protect your listening history from future disasters.
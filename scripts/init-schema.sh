#!/bin/bash

################################################################################
# Aurora Blue-Green Deployment Lab - Schema Initialization Script
#
# Purpose: Initialize the Aurora database with 12,000 tables to simulate
#          production-scale metadata overhead and test Blue-Green deployment
#          performance with large schemas.
#
# Usage: ./init-schema.sh [OPTIONS]
#
# Options:
#   --endpoint <endpoint>      Aurora cluster writer endpoint (required)
#   --database <database>      Database name (default: lab_db)
#   --username <username>      Database username (default: admin)
#   --password <password>      Database password (required)
#   --tables <count>           Number of tables to create (default: 12000)
#   --batch-size <size>        Tables per batch (default: 100)
#   --parallel <count>         Parallel connections (default: 4)
#   --help                     Show this help message
#
# Example:
#   ./init-schema.sh \
#     --endpoint my-cluster.cluster-xxxxx.us-east-1.rds.amazonaws.com \
#     --database lab_db \
#     --username admin \
#     --password MySecurePassword123 \
#     --tables 12000
#
################################################################################

# Note: set -e is NOT used because it conflicts with background jobs and error handling
# in the create_batch functions. Errors are handled explicitly instead.

# Default values
DATABASE_NAME="lab_db"
USERNAME="admin"
TABLE_COUNT=12000
BATCH_SIZE=100
PARALLEL_CONNECTIONS=4
LOG_FILE="schema-init.log"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --endpoint)
            ENDPOINT="$2"
            shift 2
            ;;
        --database)
            DATABASE_NAME="$2"
            shift 2
            ;;
        --username)
            USERNAME="$2"
            shift 2
            ;;
        --password)
            PASSWORD="$2"
            shift 2
            ;;
        --tables)
            TABLE_COUNT="$2"
            shift 2
            ;;
        --batch-size)
            BATCH_SIZE="$2"
            shift 2
            ;;
        --parallel)
            PARALLEL_CONNECTIONS="$2"
            shift 2
            ;;
        --help)
            grep "^#" "$0" | grep -v "^#!/" | sed 's/^# \?//'
            exit 0
            ;;
        *)
            echo -e "${RED}Error: Unknown option: $1${NC}"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Validate required parameters
if [ -z "$ENDPOINT" ]; then
    echo -e "${RED}Error: Aurora endpoint is required (--endpoint)${NC}"
    exit 1
fi

if [ -z "$PASSWORD" ]; then
    echo -e "${RED}Error: Database password is required (--password)${NC}"
    exit 1
fi

# Check if mysql client is installed
if ! command -v mysql &> /dev/null; then
    echo -e "${RED}Error: mysql client is not installed${NC}"
    echo ""
    echo "Install it with one of the following commands:"
    echo ""
    echo "  Amazon Linux 2023:"
    echo "    sudo yum install mariadb105 -y"
    echo ""
    echo "  Amazon Linux 2:"
    echo "    sudo yum install mysql -y"
    echo ""
    echo "  Ubuntu/Debian:"
    echo "    sudo apt-get install mysql-client -y"
    echo ""
    exit 1
fi

# Function to log messages
log() {
    local message="$1"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${timestamp} - ${message}" | tee -a "$LOG_FILE"
}

# Function to execute SQL command
execute_sql() {
    local sql="$1"
    mysql -h "$ENDPOINT" -u "$USERNAME" -p"$PASSWORD" "$DATABASE_NAME" -e "$sql" 2>&1
}

# Function to test connection without specifying database
test_connection() {
    mysql -h "$ENDPOINT" -u "$USERNAME" -p"$PASSWORD" -e "SELECT 1" 2>&1
}

# Function to create a single table
create_table() {
    local table_num=$1
    local table_name=$(printf "test_%04d" $table_num)

    local sql="CREATE TABLE IF NOT EXISTS $table_name (
        id BIGINT AUTO_INCREMENT PRIMARY KEY,
        col1 VARCHAR(255) NOT NULL,
        col2 INT DEFAULT 0,
        col3 TEXT,
        col4 DECIMAL(10,2) DEFAULT 0.00,
        col5 BIGINT DEFAULT 0,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
        INDEX idx_col1 (col1),
        INDEX idx_col2 (col2),
        INDEX idx_col5 (col5)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;"

    execute_sql "$sql" > /dev/null 2>&1

    if [ $? -eq 0 ]; then
        return 0
    else
        return 1
    fi
}

# Function to insert initial data into a table
insert_initial_data() {
    local table_num=$1
    local table_name=$(printf "test_%04d" $table_num)

    local sql="INSERT INTO $table_name (col1, col2, col3, col4, col5)
               VALUES ('initial_data', 0, 'Initial row for table $table_name', 0.00, 0)
               ON DUPLICATE KEY UPDATE col1=col1;"

    execute_sql "$sql" > /dev/null 2>&1
}

# Function to create tables in batch
create_batch() {
    local start=$1
    local end=$2
    local batch_id=$3
    local success=0
    local failed=0

    for ((i=$start; i<=$end; i++)); do
        if create_table $i; then
            ((success++))
            if [ $((i % 10)) -eq 0 ]; then
                echo -ne "\r${BLUE}Batch $batch_id: Progress $((i-start+1))/$((end-start+1)) tables created${NC}"
            fi
        else
            ((failed++))
            log "${YELLOW}Warning: Failed to create table test_$(printf "%04d" $i)${NC}"
        fi
    done

    echo -ne "\r${GREEN}Batch $batch_id: Completed - $success tables created, $failed failed${NC}\n"
    return 0
}

# Print banner
echo "================================================================================"
echo "  Aurora Blue-Green Deployment Lab - Schema Initialization"
echo "================================================================================"
echo ""
log "${BLUE}Configuration:${NC}"
log "  Endpoint: $ENDPOINT"
log "  Database: $DATABASE_NAME"
log "  Username: $USERNAME"
log "  Table Count: $TABLE_COUNT"
log "  Batch Size: $BATCH_SIZE"
log "  Parallel Connections: $PARALLEL_CONNECTIONS"
log "  Log File: $LOG_FILE"
echo "================================================================================"
echo ""

# Test database connection
log "${BLUE}Testing database connection...${NC}"
if ! test_connection > /dev/null 2>&1; then
    log "${RED}Error: Failed to connect to Aurora database${NC}"
    log "Please verify your endpoint, username, and password"
    exit 1
fi
log "${GREEN}Database connection successful${NC}"
echo ""

# Create database if not exists
log "${BLUE}Creating database if not exists...${NC}"
mysql -h "$ENDPOINT" -u "$USERNAME" -p"$PASSWORD" -e "CREATE DATABASE IF NOT EXISTS $DATABASE_NAME CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;" 2>&1 | tee -a "$LOG_FILE"
if [ ${PIPESTATUS[0]} -ne 0 ]; then
    log "${RED}Error: Failed to create database${NC}"
    exit 1
fi
log "${GREEN}Database ready${NC}"
echo ""

# Calculate batches
BATCHES=$((TABLE_COUNT / BATCH_SIZE))
if [ $((TABLE_COUNT % BATCH_SIZE)) -ne 0 ]; then
    BATCHES=$((BATCHES + 1))
fi

log "${BLUE}Starting table creation...${NC}"
log "Total batches: $BATCHES"
log "This process may take 30-60 minutes. Progress will be logged to $LOG_FILE"
echo ""

START_TIME=$(date +%s)

# Create tables in batches with parallel execution
for ((batch=0; batch<$BATCHES; batch++)); do
    start=$((batch * BATCH_SIZE + 1))
    end=$((start + BATCH_SIZE - 1))

    if [ $end -gt $TABLE_COUNT ]; then
        end=$TABLE_COUNT
    fi

    # Run batches in parallel groups
    batch_group=$((batch % PARALLEL_CONNECTIONS))

    create_batch $start $end $((batch + 1)) &

    # Wait for parallel group to complete
    if [ $batch_group -eq $((PARALLEL_CONNECTIONS - 1)) ] || [ $end -eq $TABLE_COUNT ]; then
        wait
    fi
done

wait

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
MINUTES=$((DURATION / 60))
SECONDS=$((DURATION % 60))

echo ""
log "${GREEN}Table creation completed in ${MINUTES}m ${SECONDS}s${NC}"
echo ""

# Verify table count
log "${BLUE}Verifying table count...${NC}"
ACTUAL_COUNT=$(execute_sql "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '$DATABASE_NAME' AND table_name LIKE 'test_%';" | tail -n 1)

if [ "$ACTUAL_COUNT" -eq "$TABLE_COUNT" ]; then
    log "${GREEN}Verification successful: $ACTUAL_COUNT tables created${NC}"
else
    log "${YELLOW}Warning: Expected $TABLE_COUNT tables, found $ACTUAL_COUNT tables${NC}"
fi
echo ""

# Optional: Insert initial data
read -p "Do you want to insert initial data into all tables? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    log "${BLUE}Inserting initial data...${NC}"

    for ((i=1; i<=$TABLE_COUNT; i++)); do
        insert_initial_data $i &

        if [ $((i % PARALLEL_CONNECTIONS)) -eq 0 ]; then
            wait
        fi

        if [ $((i % 100)) -eq 0 ]; then
            echo -ne "\r${BLUE}Progress: $i/$TABLE_COUNT tables populated${NC}"
        fi
    done

    wait
    echo -ne "\r${GREEN}Initial data inserted into all tables${NC}\n"
    echo ""
fi

# Display summary
echo "================================================================================"
log "${GREEN}Schema initialization completed successfully!${NC}"
echo "================================================================================"
log "Summary:"
log "  Database: $DATABASE_NAME"
log "  Tables Created: $ACTUAL_COUNT"
log "  Total Duration: ${MINUTES}m ${SECONDS}s"
log "  Log File: $LOG_FILE"
echo "================================================================================"
echo ""
log "${BLUE}Next steps:${NC}"
log "1. Deploy the workload simulator (EC2 or EKS)"
log "2. Start the write workload"
log "3. Initiate Aurora Blue-Green deployment"
log "4. Monitor the switchover behavior"
echo "================================================================================"

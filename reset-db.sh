#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🔄 Grafana Database Reset Script${NC}"
echo "=================================="

# Stop Grafana
echo -e "${YELLOW}Stopping Grafana server...${NC}"
sudo systemctl stop grafana-server
echo -e "${GREEN}✅ Grafana stopped${NC}"

# Drop and recreate public schema
echo -e "${YELLOW}Dropping public schema...${NC}"
sudo -u postgres psql -d grafana << 'EOF'
DROP SCHEMA public CASCADE;
CREATE SCHEMA public;
GRANT ALL ON SCHEMA public TO grafana;
GRANT ALL ON SCHEMA public TO public;
EOF
echo -e "${GREEN}✅ Public schema recreated${NC}"

# Start Grafana to initialize schema
echo -e "${YELLOW}Starting Grafana to initialize schema...${NC}"
sudo systemctl start grafana-server

# Wait for Grafana to be ready on port 3000
echo -e "${YELLOW}Waiting for Grafana to be ready on port 3000...${NC}"
timeout=60
elapsed=0
while ! curl -s http://localhost:3000/api/health > /dev/null 2>&1; do
    if [ $elapsed -ge $timeout ]; then
        echo -e "${RED}❌ Timeout waiting for Grafana to start${NC}"
        exit 1
    fi
    echo -n "."
    sleep 1
    elapsed=$((elapsed + 1))
done
echo ""
echo -e "${GREEN}✅ Grafana is ready${NC}"

# Wait an additional 5 seconds for migrations to complete
echo -e "${YELLOW}Waiting for database migrations to complete...${NC}"
sleep 5

# Stop Grafana
echo -e "${YELLOW}Stopping Grafana server...${NC}"
sudo systemctl stop grafana-server
echo -e "${GREEN}✅ Grafana stopped${NC}"

echo ""
echo -e "${GREEN}🎉 Database reset complete!${NC}"
echo -e "${YELLOW}You can now run the migration tool.${NC}"

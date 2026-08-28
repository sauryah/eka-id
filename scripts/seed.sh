#!/bin/bash
set -e

echo "=== Seeding EKA ID PostgreSQL Database ==="
docker exec -i eka_postgres psql -U eka_admin -d eka_id < database/seeds/dev_seeds.sql
echo "=== Seed Data Loaded Successfully ==="
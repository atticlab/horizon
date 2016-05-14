#! /usr/bin/env bash

set -e

SCENARIO=$1
CORE_SQL=./src/bitbucket.org/atticlab/horizon/test/scenarios/$SCENARIO-core.sql
HORIZON_SQL=./src/bitbucket.org/atticlab/horizon/test/scenarios/$SCENARIO-horizon.sql

echo "psql $STELLAR_CORE_DATABASE_URL < $CORE_SQL" 
psql $STELLAR_CORE_DATABASE_URL < $CORE_SQL 
echo "psql $DATABASE_URL < $HORIZON_SQL"
psql $DATABASE_URL < $HORIZON_SQL 

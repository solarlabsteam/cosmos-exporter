#!/bin/sh
# sleep 10000
/usr/local/bin/cosmos-exporter --denom "$DENOM" --denom-coefficient "$DENOM_COEFFICIENT" --bech-prefix "$BECH_PREFIX"
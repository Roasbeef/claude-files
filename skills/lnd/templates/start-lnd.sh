#!/usr/bin/env bash
# Extended start-lnd.sh with support for all backends and custom args
#
# Environment variables:
#   BACKEND     - bitcoind, btcd, or neutrino (required)
#   NETWORK     - regtest, testnet, signet, mainnet (default: depends on backend)
#   CHAIN       - bitcoin (default)
#   RPCHOST     - RPC host for bitcoind/btcd backends
#   RPCUSER     - RPC username
#   RPCPASS     - RPC password
#   RPCCRTPATH  - Path to btcd RPC cert (btcd only)
#   LND_DEBUG   - Log level (default: debug)
#   LND_EXTRA_ARGS - Additional args to pass to lnd
#   NEUTRINO_PEER - Peer for neutrino mode
#   FEE_URL     - External fee estimation URL (neutrino only)
#   NOSEEDBACKUP - Set to false to require wallet seed (default: true for testing)

set -e

error() {
    echo "$1" >&2
    exit 1
}

# Set defaults based on backend
DEFAULT_NETWORK="regtest"
if [ "$BACKEND" == "btcd" ]; then
    DEFAULT_NETWORK="simnet"
elif [ "$BACKEND" == "neutrino" ]; then
    DEFAULT_NETWORK="testnet"
fi

# Apply defaults
NETWORK="${NETWORK:-$DEFAULT_NETWORK}"
CHAIN="${CHAIN:-bitcoin}"
RPCHOST="${RPCHOST:-blockchain}"
RPCUSER="${RPCUSER:-devuser}"
RPCPASS="${RPCPASS:-devpass}"
DEBUG="${LND_DEBUG:-debug}"
NOSEEDBACKUP="${NOSEEDBACKUP:-true}"
HOSTNAME=$(hostname)

# Build base args
BASE_ARGS=""
if [ "$NOSEEDBACKUP" == "true" ]; then
    BASE_ARGS="--noseedbackup"
fi

case "$BACKEND" in
    bitcoind)
        exec lnd \
            $BASE_ARGS \
            "--$CHAIN.active" \
            "--$CHAIN.$NETWORK" \
            "--$CHAIN.node=bitcoind" \
            "--bitcoind.rpchost=$RPCHOST" \
            "--bitcoind.rpcuser=$RPCUSER" \
            "--bitcoind.rpcpass=$RPCPASS" \
            "--bitcoind.zmqpubrawblock=tcp://$RPCHOST:28332" \
            "--bitcoind.zmqpubrawtx=tcp://$RPCHOST:28333" \
            "--rpclisten=$HOSTNAME:10009" \
            "--rpclisten=localhost:10009" \
            "--restlisten=0.0.0.0:8080" \
            "--debuglevel=$DEBUG" \
            $LND_EXTRA_ARGS \
            "$@"
        ;;
    btcd)
        RPCCRTPATH="${RPCCRTPATH:-/rpc/rpc.cert}"
        exec lnd \
            $BASE_ARGS \
            "--$CHAIN.active" \
            "--$CHAIN.$NETWORK" \
            "--$CHAIN.node=btcd" \
            "--btcd.rpccert=$RPCCRTPATH" \
            "--btcd.rpchost=$RPCHOST" \
            "--btcd.rpcuser=$RPCUSER" \
            "--btcd.rpcpass=$RPCPASS" \
            "--rpclisten=$HOSTNAME:10009" \
            "--rpclisten=localhost:10009" \
            "--restlisten=0.0.0.0:8080" \
            "--debuglevel=$DEBUG" \
            $LND_EXTRA_ARGS \
            "$@"
        ;;
    neutrino)
        NEUTRINO_PEER="${NEUTRINO_PEER:-faucet.lightning.community}"
        FEE_URL="${FEE_URL:-https://nodes.lightning.computer/fees/v1/btc-fee-estimates.json}"
        exec lnd \
            $BASE_ARGS \
            "--$CHAIN.active" \
            "--$CHAIN.$NETWORK" \
            "--$CHAIN.node=neutrino" \
            "--neutrino.connect=$NEUTRINO_PEER" \
            "--feeurl=$FEE_URL" \
            "--rpclisten=$HOSTNAME:10009" \
            "--rpclisten=localhost:10009" \
            "--restlisten=0.0.0.0:8080" \
            "--debuglevel=$DEBUG" \
            $LND_EXTRA_ARGS \
            "$@"
        ;;
    *)
        error "Unknown backend: $BACKEND. Use bitcoind, btcd, or neutrino."
        ;;
esac

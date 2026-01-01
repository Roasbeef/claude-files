# LND CLI Reference

Complete reference for `lncli` commands. All functionality available via gRPC/REST is accessible through the CLI.

## Node & Wallet

```bash
# Node information
lncli getinfo                    # Node status, block height, peers, channels
lncli getrecoveryinfo            # Recovery mode status
lncli debuglevel --level=debug   # Set log level

# Wallet balance
lncli walletbalance              # On-chain balance
lncli channelbalance             # Lightning channel balance

# Addresses
lncli newaddress p2wkh           # Native SegWit (recommended)
lncli newaddress np2wkh          # Nested SegWit
lncli newaddress p2tr            # Taproot

# On-chain send
lncli sendcoins --addr=<address> --amt=<sats>
lncli sendcoins --addr=<address> --amt=<sats> --conf_target=6
lncli sendcoins --addr=<address> --sweepall  # Send all funds

# Transactions
lncli listchaintxns              # List on-chain transactions
```

## Peers

```bash
# Connect/disconnect
lncli connect <pubkey>@<host>:<port>
lncli disconnect <pubkey>

# List peers
lncli listpeers
lncli listpeers | jq '.peers[] | {pubkey: .pub_key[:16], address: .address}'
```

## Channels

```bash
# Open channel
lncli openchannel --node_key=<pubkey> --local_amt=<sats>
lncli openchannel --node_key=<pubkey> --local_amt=<sats> --push_amt=<sats>
lncli openchannel --node_key=<pubkey> --local_amt=<sats> --private  # Private channel
lncli openchannel --node_key=<pubkey> --local_amt=<sats> --conf_target=3

# List channels
lncli listchannels               # Active channels
lncli pendingchannels            # Pending open/close
lncli closedchannels             # Closed channels

# Channel info
lncli getchaninfo <chan_id>      # Get channel from graph

# Close channel
lncli closechannel --funding_txid=<txid> --output_index=<idx>
lncli closechannel --funding_txid=<txid> --output_index=<idx> --force
lncli closechannel --chan_point=<txid:idx>   # Alternative format

# Channel balance details
lncli listchannels | jq '.channels[] | {
  peer: .remote_pubkey[:16],
  capacity: .capacity,
  local: .local_balance,
  remote: .remote_balance
}'
```

## Invoices (Receiving)

```bash
# Create invoice
lncli addinvoice --amt=<sats>
lncli addinvoice --amt=<sats> --memo="description"
lncli addinvoice --amt=<sats> --expiry=3600   # 1 hour expiry
lncli addinvoice --amt_msat=<msats>           # Millisatoshi precision

# List invoices
lncli listinvoices
lncli listinvoices --pending_only

# Lookup invoice
lncli lookupinvoice <r_hash>
lncli lookupinvoice --pay_addr=<payment_addr>

# Decode any invoice
lncli decodepayreq <bolt11>
```

## Payments (Sending)

```bash
# Pay invoice
lncli payinvoice --pay_req=<bolt11>
lncli payinvoice --pay_req=<bolt11> --fee_limit=100  # Max fee in sats
lncli payinvoice --pay_req=<bolt11> --timeout=60s

# Keysend (spontaneous payment)
lncli sendpayment --dest=<pubkey> --amt=<sats> --keysend

# Query route (without paying)
lncli queryroutes --dest=<pubkey> --amt=<sats>

# List payments
lncli listpayments
lncli listpayments --include_incomplete

# Track payment status
lncli trackpayment <payment_hash>
```

## Routing

```bash
# Fee report (your channels)
lncli feereport

# Update channel policy
lncli updatechanpolicy --base_fee_msat=1000 --fee_rate=0.000001 --time_lock_delta=40
lncli updatechanpolicy --chan_point=<txid:idx> --base_fee_msat=500  # Single channel

# Forwarding history
lncli fwdinghistory
lncli fwdinghistory --start_time=-1d  # Last 24 hours

# Mission control (routing intelligence)
lncli querymc               # Query state
lncli resetmc               # Reset routing memory
lncli getmccfg              # Get config
```

## Network Graph

```bash
# Describe graph
lncli describegraph

# Get node info
lncli getnodeinfo <pubkey>
lncli getnodeinfo <pubkey> --include_channels

# Get channel from graph
lncli getchaninfo <chan_id>

# Get network info
lncli getnetworkinfo
```

## Wallet Management

```bash
# Wallet operations
lncli wallet accounts list
lncli wallet estimatefee --conf_target=6

# UTXOs
lncli listunspent
lncli listunspent --min_confs=1 --max_confs=100

# Bump fee (CPFP)
lncli wallet bumpfee <outpoint>

# Label transaction
lncli wallet labeltx --txid=<txid> --label="channel open"
```

## Backups

```bash
# Export all channel backups
lncli exportchanbackup --all

# Export single channel
lncli exportchanbackup --chan_point=<txid:idx>

# Verify backup
lncli verifychanbackup --multi_file=backup.file

# Restore from backup (requires wallet unlock)
lncli restorechanbackup --multi_file=backup.file
```

## Signing & Messages

```bash
# Sign message
lncli signmessage --msg="Hello World"

# Verify message
lncli verifymessage --msg="Hello World" --sig=<signature>
```

## Watchtower

```bash
# Client (use external watchtower)
lncli wtclient towers           # List connected towers
lncli wtclient add <pubkey>@<host>:<port>
lncli wtclient remove <pubkey>
lncli wtclient stats

# Server (run a watchtower)
lncli tower info
```

## Useful Command Combinations

```bash
# Total channel capacity
lncli listchannels | jq '[.channels[].capacity | tonumber] | add'

# Active channel count
lncli listchannels | jq '[.channels[] | select(.active)] | length'

# Get your pubkey
lncli getinfo | jq -r '.identity_pubkey'

# Wait for invoice payment
lncli subscribeinvoices

# Watch channel events
lncli subscribechannelevents

# Check if synced
lncli getinfo | jq '.synced_to_chain'
```

## Common Flags

| Flag | Description |
|------|-------------|
| `--network` | regtest, testnet, signet, mainnet |
| `--rpcserver` | Host:port of lnd RPC |
| `--macaroonpath` | Path to macaroon file |
| `--tlscertpath` | Path to TLS cert |
| `-j, --json` | Output as JSON |

## Exit Codes

- `0` - Success
- `1` - General error
- `2` - Invalid arguments

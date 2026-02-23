# lnget Skill

Use lnget to fetch resources that require L402 Lightning payments.

## When to Use

Use lnget when:
- Fetching resources from L402-protected APIs
- Downloading files behind Lightning paywalls
- Making requests that may require micropayments

## Quick Reference

### Basic Fetch
```bash
# Fetch and print to stdout
lnget https://api.example.com/data

# Save to file
lnget -o output.json https://api.example.com/data

# Quiet mode
lnget -q https://api.example.com/data | jq .
```

### Payment Control
```bash
# Set max payment (default: 1000 sats)
lnget --max-cost 500 https://api.example.com/data

# Set max routing fee (default: 10 sats)
lnget --max-fee 5 https://api.example.com/data

# Don't auto-pay (just show 402 response)
lnget --no-pay https://api.example.com/data
```

### Output Formats
```bash
# JSON output (default)
lnget https://api.example.com/data

# Human-readable output
lnget --human https://api.example.com/data
```

### Token Management
```bash
# List cached tokens
lnget tokens list

# Remove expired/unused token
lnget tokens remove example.com

# Clear all tokens
lnget tokens clear --force
```

### Backend Status
```bash
# Check connection status
lnget ln status

# Get node info
lnget ln info
```

## Common Patterns

### Fetch JSON API Data
```bash
# Fetch and parse JSON
data=$(lnget -q https://api.example.com/data)
echo "$data" | jq '.result'
```

### Download File with Progress
```bash
lnget -o file.zip https://api.example.com/file.zip
```

### Resume Partial Download
```bash
lnget -c -o large.zip https://api.example.com/large.zip
```

### Check if Token Exists
```bash
if lnget tokens show example.com >/dev/null 2>&1; then
  echo "Token cached"
fi
```

## Configuration

Config file: `~/.lnget/config.yaml`

Key settings:
- `l402.max_cost_sats`: Maximum automatic payment (default: 1000)
- `l402.max_fee_sats`: Maximum routing fee (default: 10)
- `ln.mode`: Backend type (lnd, lnc, neutrino)
- `output.format`: Default output format (json, human)

## Exit Codes

- 0: Success
- 1: General error
- 2: Payment exceeded max cost
- 3: Payment failed
- 4: Network error

## Notes

- Tokens are cached per-domain automatically
- JSON output is default (for agent consumption)
- Use `--human` for human-readable output
- Configure lnd connection in `~/.lnget/config.yaml`

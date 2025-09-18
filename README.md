# Physical Devices Monitor

A simple console app that  shows device  status PT NGFW in real-time.

## What it does

- Polls your device API every 5 seconds (configurable)
- Shows devices grouped by logical device with pretty colors
- If there is a problem with the connection (displays the latest known data)
- Auto-reconnects when auth expires

## Quick start

```bash
# Build it
go build . -o pt_device_monitor

# Run it (uses default endpoint)
./pt_device_monitor
```

### custom settings

#### parameters
```
./pt_device_monitor -base_url https://your-mgmt.local/api/v2/ -interval 1m -username username -password password
```

#### Environment variables
```bash
export PT_BASE_URL="https://my-api.com/api/v2/"
export PT_API_USERNAME="admin"
export PT_API_PASSWORD="admin"
export PT_POLL_INTERVAL="1m"
./pt_mgmt
```


## Options

```
-base_url    Url PT MGMT for API (REQUIRED) (env: PT_BASE_URL) (example: https://your-mgmt.local/api/v2/)
-username    username for api authentication (env: PT_API_USERNAME)  (default: admin)
-password    password for api authentication (env: PT_API_PASSWORD)  (default: admin) 
-interval    How often to poll  (env: PT_API_PASSWORD)               (default: 5s)
```

## What you'll see

```console
┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Physical Devices Monitor - Last Updated: 2025-09-18 11:22:22 (Total: 3)                                             │                                     ├─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ LOGICAL DEVICE: my-ngfw-1 (STANDALONE) - Contexts: Default (default)                                                │
│  └─  my-ngfw-1                   │ pt-ngfw-1010       │ DISCONNECTED       │ 1.23.4.5     │ -             │ 1.8.0   │  │                                                                                                                     │
│ LOGICAL DEVICE: pt-ngfw-cluster (ACTIVE_STANDBY) - Contexts: Default (default), test                                │
│  ├─  ngfw-node1 [UNSPECIFIED]    │ pt-ngfw-2020       │ DISCONNECTED       │ 2.3.4.5      │ Priority: 5   │ 1.8.0   │
│  └─  ngfw-node2 [UNSPECIFIED]    │ pt-ngfw-2020       │ DISCONNECTED       │ 6.7.8.9      │ Priority: 6   │ 1.8.0   │ ├─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Poll Interval: 5s │ Press Ctrl+C to exit │ MGMT: 10.10.10.10                                                        │                                     └─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```



That's it! Press Ctrl+C to exit.
# Physical Devices Monitor

A simple console app that  shows device  status PT NGFW in real-time.
<img width="1391" height="282" alt="image" src="https://github.com/user-attachments/assets/b0a653d0-01b0-487c-b94a-cb1b9d43213f" />

## What it does

- Polls your device API every 5 seconds (configurable)
- Shows devices grouped by logical device with pretty colors
- If there is a problem with the connection (displays the latest known data)
- Auto-reconnects when auth expires

## Quick start

```bash
git clone https://github.com/yurgers/pt_device_monitor
cd pt_device_monitor

# Build it
 go build -o pt_device_monitor .

# Run it (uses default endpoint)
./pt_device_monitor -base_url https://your-mgmt.local/api/v2/ 
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


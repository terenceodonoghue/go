# solar

Background service that polls a Fronius Primo 8.2-1 solar inverter every 5 seconds and writes metrics to InfluxDB 2.x for display in Grafana.

## Setup

### Prerequisites

- Go 1.26+
- Docker + Docker Compose
- A Fronius inverter accessible on the local network
- An InfluxDB 2.x instance

### Local development

1. Start InfluxDB:

   ```sh
   docker compose up -d
   ```

2. Open the InfluxDB UI at http://localhost:8086 and log in with `admin` / `adminpassword`.

3. Create a write token scoped to the `solar-raw` bucket:

   - Go to **Load Data → API Tokens → Generate API Token → Custom API Token**
   - Grant **Write** access to `solar-raw`
   - Copy the token

4. Create the `solar-hourly` and `solar-daily` buckets (see [InfluxDB setup](#influxdb-setup)).

5. Copy `.env` and fill in your values:

   ```sh
   cp .env .env.local
   ```

   Set `INVERTER_URL` to your inverter's IP, `INVERTER_CAPACITY_W` to its rated output in watts, and `INFLUX_TOKEN` to the token from step 3.

6. Run the service:

   ```sh
   set -a && source .env.local && set +a
   go run .
   ```

## Environment variables

| Variable | Description |
|----------|-------------|
| `INVERTER_URL` | Inverter base URL, e.g. `http://192.168.1.100` |
| `INVERTER_CAPACITY_W` | Rated output in watts; used to compute `utilisation` (e.g. `8200`) |
| `INFLUX_URL` | InfluxDB base URL, e.g. `http://localhost:8086` |
| `INFLUX_TOKEN` | InfluxDB API token with write access to the raw bucket |
| `INFLUX_ORG` | InfluxDB organisation; must match `DOCKER_INFLUXDB_INIT_ORG` |
| `INFLUX_BUCKET` | Target bucket; must match `DOCKER_INFLUXDB_INIT_BUCKET` |

All variables are required — the service will not start if any are missing.

## Hardcoded values

| Value | Setting | Notes |
|-------|---------|-------|
| Poll interval | `5 s` | Fronius minimum is ~4 s |
| Backoff max | `10 min` | Max wait when inverter is unreachable |
| Archive interval | `24 h` | How often month energy is refreshed from the archive API |
| Health address | `:8082` | Used by Docker `healthcheck` |
| Device ID tag | `fronius` | `device_id` tag on every InfluxDB point |

## InfluxDB setup

### Buckets

Create three buckets in the InfluxDB UI (**Load Data → Buckets → Create Bucket**):

| Bucket | Retention | Purpose |
|--------|-----------|---------|
| `solar-raw` | 30 days | Raw 5-second writes from this service (auto-created in local dev) |
| `solar-hourly` | 365 days | 1-hour aggregates produced by a scheduled Task |
| `solar-daily` | Never | 1-day aggregates, permanent record |

### Tokens

Create two API tokens (**Load Data → API Tokens**):

- **Write token** (used by this service): Custom token with **Write** on `solar-raw` only.
- **Read token** (used by Grafana): Custom token with **Read** on all three buckets.

### Downsampling tasks

Create the following Tasks in the InfluxDB UI (**Tasks → Create Task**). Each task runs on a schedule and aggregates from the finer-grained bucket into the coarser one. Instantaneous measurements (power, voltage, current, frequency) use `mean`; cumulative energy counters use `last` so Grafana can compute daily increments via `increase()`.

#### Task 1: Raw → hourly (runs every hour)

```flux
option task = {name: "solar-downsample-hourly", every: 1h, offset: 5m}

from(bucket: "solar-raw")
  |> range(start: -task.every)
  |> filter(fn: (r) => r._measurement == "inverter")
  |> filter(fn: (r) =>
      r._field == "pac" or
      r._field == "pac_kw" or
      r._field == "iac" or
      r._field == "uac" or
      r._field == "fac" or
      r._field == "idc" or
      r._field == "udc" or
      r._field == "utilisation")
  |> aggregateWindow(every: 1h, fn: mean, createEmpty: false)
  |> to(bucket: "solar-hourly", org: "homelab")

from(bucket: "solar-raw")
  |> range(start: -task.every)
  |> filter(fn: (r) => r._measurement == "inverter")
  |> filter(fn: (r) =>
      r._field == "day_energy" or
      r._field == "year_energy" or
      r._field == "total_energy")
  |> aggregateWindow(every: 1h, fn: last, createEmpty: false)
  |> to(bucket: "solar-hourly", org: "homelab")
```

#### Task 2: Hourly → daily (runs every day)

```flux
option task = {name: "solar-downsample-daily", every: 1d, offset: 10m}

from(bucket: "solar-hourly")
  |> range(start: -task.every)
  |> filter(fn: (r) => r._measurement == "inverter")
  |> filter(fn: (r) =>
      r._field == "pac" or
      r._field == "pac_kw" or
      r._field == "iac" or
      r._field == "uac" or
      r._field == "fac" or
      r._field == "idc" or
      r._field == "udc" or
      r._field == "utilisation")
  |> aggregateWindow(every: 1d, fn: mean, createEmpty: false)
  |> to(bucket: "solar-daily", org: "homelab")

from(bucket: "solar-hourly")
  |> range(start: -task.every)
  |> filter(fn: (r) => r._measurement == "inverter")
  |> filter(fn: (r) =>
      r._field == "day_energy" or
      r._field == "year_energy" or
      r._field == "total_energy")
  |> aggregateWindow(every: 1d, fn: last, createEmpty: false)
  |> to(bucket: "solar-daily", org: "homelab")
```

## Data model

**Measurement:** `inverter`

**Tags:**

| Tag | Example | Description |
|-----|---------|-------------|
| `device_id` | `fronius` | Identifies the inverter brand (hardcoded) |

**Fields:**

| Field | Unit | Description |
|-------|------|-------------|
| `pac` | W | AC power output (instantaneous) |
| `pac_kw` | kW | AC power output (`pac / 1000`) |
| `iac` | A | AC current |
| `uac` | V | AC voltage |
| `fac` | Hz | AC frequency |
| `idc` | A | DC current from panels |
| `udc` | V | DC voltage from panels |
| `day_energy` | Wh | Energy produced today (resets at midnight) |
| `year_energy` | Wh | Energy produced this year |
| `total_energy` | Wh | Lifetime energy produced |
| `utilisation` | % | Output as a percentage of rated capacity |

No data is written when the inverter is not producing (e.g. at night). Gaps in the time series are intentional.

## Technical design

### Polling

The service polls the Fronius Solar API's `GetInverterRealtimeData.cgi` endpoint — the only endpoint available on a non-hybrid single-phase inverter without a smart meter. No authentication is required.

```
GET http://<INVERTER_URL>/solar_api/v1/GetInverterRealtimeData.cgi
    ?Scope=Device&DeviceId=1&DataCollection=CommonInverterData
```

### Backoff

The inverter shuts down overnight. Two distinct behaviours handle this:

- **Inverter responds but isn't producing** (`StatusCode != 7`, e.g. during startup or shutdown): the poll is a no-op — no write, no error, polling continues at the normal interval.
- **Inverter unreachable** (network error or timeout): exponential backoff doubles the wait on each consecutive failure, capped at `POLL_BACKOFF_MAX` (default 5 minutes). Normal polling resumes — and a recovery message is logged — on the first successful fetch.

### Writes

Each successful poll produces one InfluxDB point with the fields listed above. Writes use the blocking API so errors are surfaced immediately in logs. `pac_kw` and `utilisation` are computed before writing so Grafana dashboards can use them as raw fields without Flux transforms.

Timestamps use the poller's wall clock (`time.Now()`) rather than the inverter's `Head.Timestamp`, which may drift and carries timezone offset strings that complicate parsing.

### Graceful shutdown

A `signal.NotifyContext` propagates `SIGINT`/`SIGTERM` through the poll loop and any in-flight InfluxDB write. The health check server on `:8082` starts before the poll loop and is used by Docker's `healthcheck` directive.

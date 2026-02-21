# go

Go microservices, each independently deployable as a Docker container and published to GitHub Container Registry.

## Services

### [auth](services/auth/)

Passwordless authentication API built on WebAuthn/passkeys. Users register and sign in using a passkey (Touch ID, Face ID, or a hardware key) — no passwords stored, with sessions kept in Redis.

### [solar](services/solar/)

Background service that polls a Fronius solar inverter every 5 seconds and writes real-time metrics to InfluxDB.

# Backend for processing the preregistration process

Used only for creating a single point of communication between services, stores no data except for service credentials.

## Environment variables
To run you must provide these env variables:
- SIB_KEY
- SIB_ENDPOINT

## Running
```bash
SIB_KEY=key SIB_ENDPOINT=https://sib.endpoint.com go run .
```

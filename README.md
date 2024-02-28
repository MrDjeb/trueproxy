## Getting Started
```bash
docker pull mrdjeb/trueproxy:latest && docker run -p 8080:62801 -p 8000:62802 --rm mrdjeb/trueproxy:latest
```

## Usage
```bash
curl --ssl-no-revoke -x localhost:8080 https://mail.ru
curl -x localhost:8080 http://mail.ru
```

- requests
- request/:id
- repeat/:id
- scan/:id

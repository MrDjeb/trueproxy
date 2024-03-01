## Getting Started
```bash
docker pull mrdjeb/trueproxy:latest && docker run -p 8080:62801 -p 8000:62802 --rm mrdjeb/trueproxy:latest
```

## Usage
```bash
curl --ssl-no-revoke -x localhost:8080 https://mail.ru
curl -x localhost:8080 http://mail.ru

curl -X POST -H "Content-Type: application/json" -d '{"productId": 123456, "quantity": 100}'

curl -x localhost:62801 -XPOST -d 'wqefq3fq3fqef' --ssl-no-revoke  https://mail.ru 
```

- requests
- request/:id
- repeat/:id
- scan/:id

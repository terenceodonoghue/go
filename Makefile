.PHONY: setup security-scan security-scan-auth-api security-scan-fron-svc

setup:
	brew bundle
	pre-commit install

security-scan: security-scan-auth-api security-scan-fron-svc

security-scan-auth-api:
	docker build -t auth-api:scan -f services/auth-api/Dockerfile .
	trivy image --severity CRITICAL,HIGH --ignore-unfixed --exit-code 1 auth-api:scan

security-scan-fron-svc:
	docker build -t fron-svc:scan -f services/fron-svc/Dockerfile .
	trivy image --severity CRITICAL,HIGH --ignore-unfixed --exit-code 1 fron-svc:scan

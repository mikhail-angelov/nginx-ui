include .env

test:
	go test -v ./...
	
build:
	GOOS=linux GOARCH=amd64 go build -o dist/nginx-ui ./app

build-local:
	GOOS=darwin GOARCH=arm64 go build -o dist/nginx-ui ./app

run-local:
	./dist/nginx-ui -configDir=./app/temp -dev=true -docker=true -email=test@test.com -pass=1234 -port=3005

run:
	./nginx-ui -configDir=/etc/nginx -email=test@test.com -pass=1234 -port=3005

install-nginx:
	@echo "Checking if Nginx is installed on $(REMOTE_HOST)..."
	@ssh $(REMOTE_HOST) 'if ! command -v nginx &> /dev/null; then \
		echo "Nginx is not installed. Installing Nginx..."; \
		sudo apt-get update; \
		sudo apt-get install -y nginx; \
		ssh root@${HOST} "mkdir -p /etc/nginx/cong"; \
		ssh root@${HOST} "mkdir -p /etc/nginx/certs"; \
		echo "Nginx installed successfully. ✅"; \
	else \
		echo "Nginx is already installed. ✅"; \
	fi'
install-supervisor:
	@echo "Checking if Supervisor is installed on $(REMOTE_HOST)..."
	@ssh $(REMOTE_HOST) 'if ! command -v supervisord &> /dev/null; then \
		echo "Supervisor is not installed. Installing Supervisor..."; \
		sudo apt-get install -y supervisor; \
		echo "Supervisor installed successfully. ✅"; \
	else \
		echo "Supervisor is already installed. ✅"; \
	fi'

initial-deploy: build install-nginx install-supervisor
	-ssh ${REMOTE_HOST} "mkdir -p /opt/ngx"
	-scp -r dist/nginx-ui ${REMOTE_HOST}:/opt/ngx
	@EMAIL=${EMAIL} PASS=${PASS} envsubst < ./supervisior/ngx.conf > ./dist/ngx.conf
	-scp -r ./dist/ngx.conf ${REMOTE_HOST}:/etc/supervisor/conf.d/ngx.conf
	-ssh ${REMOTE_HOST} "supervisorctl reread; supervisorctl update; supervisorctl start ngx"
	@echo "Initial deploying Nginx UI to $(REMOTE_HOST) is done. ✅"
	
deploy:
	scp -r dist/nginx-ui ${REMOTE_HOST}:/opt/ngx

.PHONY: build build-local run-local initial-deploy install-nginx install-supervisor deploy
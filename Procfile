workerserver: go run -tags=$GO_BUILD_TAGS src/ee/workerserver/main.go
hosting: go run -tags=$GO_BUILD_TAGS src/ee/hosting/main.go
ui: node -e "setTimeout(()=>{},5000)" && cd ./src/ui && npm install && npm run dev -- --port 5400
www: cd ./src/www && npm install && npm run dev -- --port 5500

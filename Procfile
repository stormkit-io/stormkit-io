workerserver: go run -tags=$GO_BUILD_TAGS src/ee/workerserver/main.go
hosting: go run -tags=$GO_BUILD_TAGS src/ee/hosting/main.go
ui: sleep 5;cd ./src/ui && npm install && npm run dev -- --port 5400

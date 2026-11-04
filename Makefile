all: app

SAM := sam
REGION := ap-northeast-1
BUCKET := nana-lambda

STACK_NAME := momochi

app:
	go build

app-for-deploy:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/bootstrap

deploy: app-for-deploy
	$(SAM) deploy --region $(REGION) --s3-bucket $(BUCKET) --capabilities CAPABILITY_IAM --template-file template.yaml --stack-name $(STACK_NAME)

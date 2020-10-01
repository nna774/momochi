all: app

app:
	go build

deploy: app
	sam package --template-file template.yaml --s3-bucket nana-lambda --output-template-file packaged-template.yml
	sam deploy --template-file packaged-template.yml --capabilities CAPABILITY_IAM --region ap-northeast-1 --stack-name momochi

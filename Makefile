PROJECT:=$(shell go list -m)
TAG:=$(shell date +%Y%m%d-%H%M)

include .env
AWS_ECR_REPOSITORY_BASE=$(AWS_ACCOUNT).dkr.ecr.$(AWS_REGION).amazonaws.com

define MAKEFILE_LAMBDA_ASSUME_ROLE_JSON
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": "sts:AssumeRole",
            "Effect": "Allow",
            "Principal": {
                "Service": "lambda.amazonaws.com"
            }
        }
    ]
}
endef
export MAKEFILE_LAMBDA_ASSUME_ROLE_JSON

define MAKEFILE_LAMBDA_POLICY
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": "logs:CreateLogGroup",
            "Resource": "arn:aws:logs:$(AWS_REGION):$(AWS_ACCOUNT):*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "logs:CreateLogStream",
                "logs:PutLogEvents"
            ],
            "Resource": [
                "arn:aws:logs:$(AWS_REGION):$(AWS_ACCOUNT):log-group:/aws/lambda/$(LAMBDA_FUNCTION):*"
            ]
        },
		{
			"Effect": "Allow",
			"Action": [
				"secretsmanager:GetSecretValue",
				"secretsmanager:ListSecrets",
				"secretsmanager:DescribeSecret"
			],
			"Resource": [
				"arn:aws:secretsmanager:ap-northeast-1:384081048358:secret:SlackSecret-Vzss8J"
			]
		},
        {
            "Effect": "Allow",
            "Action": [
                "sns:GetTopicAttributes",
                "sns:List*"
            ],
            "Resource": "*"
        }
    ]
}
endef
export MAKEFILE_LAMBDA_POLICY

check-parameter:
ifndef PACKAGE
		@echo variable PACKAGE is not defined.
		@exit 1
endif
	@echo parameter is checked.

.PHONY: build
build:
	@echo $(PACKAGE) $(AWS_ECR_REPOSITORY_BASE) && \
	aws ecr get-login-password --region $(AWS_REGION) | docker login --username AWS --password-stdin $(AWS_ECR_REPOSITORY_BASE)/$(PACKAGE) && \
	docker build -t $(PACKAGE):$(TAG) --build-arg MODULE=$(PACKAGE) . && \
	docker tag $(PACKAGE):$(TAG) $(AWS_ECR_REPOSITORY_BASE)/$(PACKAGE):$(TAG) && \
	docker push $(AWS_ECR_REPOSITORY_BASE)/$(PACKAGE):$(TAG)

update-lambda-function:
	@echo "current lambda image URI: $(shell aws lambda get-function --function-name $(LAMBDA_FUNCTION) | jq -r .Code.ImageUri)" && \
	aws lambda update-function-code --function-name "$(LAMBDA_FUNCTION)" --image-uri $(AWS_ECR_REPOSITORY_BASE)/$(PACKAGE):$(TAG)

create-lambda-function:
ifndef LAMBDA_FUNCTION
		@echo variable PACKAGE is not defined.
		@exit 1
endif
	@echo "Create Role..." && \
	(aws iam get-role --role-name "Lambda$(LAMBDA_FUNCTION)" || \
		aws iam create-role --role-name "Lambda$(LAMBDA_FUNCTION)" --assume-role-policy-document "$$MAKEFILE_LAMBDA_ASSUME_ROLE_JSON") && \
	echo "Create Policy..." && \
	MAKEFILE_POLICY=`(aws iam list-policies | jq '.Policies[] | select(.PolicyName == "Lambda$(LAMBDA_FUNCTION)")' -e || \
		aws iam create-policy --policy-name "Lambda$(LAMBDA_FUNCTION)" --policy-document "$$MAKEFILE_LAMBDA_POLICY" | jq '.Policy')` && \
	echo "$$MAKEFILE_POLICY" && \
	MAKEFILE_POLICY_ARN=`echo $$MAKEFILE_POLICY | jq '.Arn' -r` && \
	echo "Attach Policy..." && \
	(aws iam list-attached-role-policies --role-name "Lambda$(LAMBDA_FUNCTION)" | jq '.AttachedPolicies[] | select(.PolicyName == "Lambda$(LAMBDA_FUNCTION)")' -e || \
		aws iam attach-role-policy --role-name "Lambda$(LAMBDA_FUNCTION)" --policy-arn "$$MAKEFILE_POLICY_ARN") && \
	echo "Create Lambda Function..." && \
	(aws lambda get-function --function-name "$(LAMBDA_FUNCTION)" || \
		aws lambda create-function --function-name "$(LAMBDA_FUNCTION)" --role "$(shell aws iam get-role --role-name "Lambda$(LAMBDA_FUNCTION)" | jq '.Role.Arn' -r)" --region "$(AWS_REGION)" --package-type "Image" --code "ImageUri=$(AWS_ECR_REPOSITORY_BASE)/$(PACKAGE):$(TAG)")

create-ecr-repository:
ifndef PACKAGE
		@echo variable PACKAGE is not defined.
		@exit 1
endif
	@(echo "Create ECR Repository..." && aws ecr describe-repositories | jq '.repositories[] | select(.repositoryName == "$(PACKAGE)")' -e || \
		aws ecr create-repository --repository-name "$(PACKAGE)" --image-tag-mutability IMMUTABLE)



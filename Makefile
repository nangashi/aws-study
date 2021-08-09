PROJECT:=$(shell go list -m)
TAG:=$(shell date +%Y%m%d-%H%M)

include .env
AWS_ECR_REPOSITORY_BASE=$(AWS_ACCOUNT).dkr.ecr.$(AWS_REGION).amazonaws.com

define MAKEFILE_LAMBDA_ASSUME_ROLE_JSON
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": "sts:AssumeRole",
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
				"arn:aws:secretsmanager:$(AWS_REGION):$(AWS_ACCOUNT):secret:SlackSecret-Vzss8J"
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

non-parameter:
ifndef PACKAGE
		@echo variable PACKAGE is not defined.
		@exit 1
endif
	@echo parameter is checked.

.PHONY: build
build:
	aws ecr get-login-password --region $(AWS_REGION) | docker login --username AWS --password-stdin $(AWS_ECR_REPOSITORY_BASE)/$(PACKAGE) && \
	docker build -t $(PACKAGE):$(TAG) --build-arg MODULE=$(PACKAGE) . && \
	docker tag $(PACKAGE):$(TAG) $(AWS_ECR_REPOSITORY_BASE)/$(PACKAGE):$(TAG) && \
	docker push $(AWS_ECR_REPOSITORY_BASE)/$(PACKAGE):$(TAG)

update-lambda-function:
	@echo "* update lambda function..."
	@echo "* current lambda image URI: $(shell aws lambda get-function --function-name $(LAMBDA_FUNCTION) | jq -r .Code.ImageUri)"
	aws lambda update-function-code --function-name "$(LAMBDA_FUNCTION)" --image-uri $(AWS_ECR_REPOSITORY_BASE)/$(PACKAGE):$(TAG)
	@echo "done."

create-lambda-function:
ifndef LAMBDA_FUNCTION
		@echo variable PACKAGE is not defined.
		@exit 1
endif
	@echo "* checking role..."
	aws iam get-role --role-name "Lambda$(LAMBDA_FUNCTION)" > /dev/null 2>&1 && echo "* role is already created." || \
		(echo "* role is not created. creating..." && \
		aws iam create-role --role-name "Lambda$(LAMBDA_FUNCTION)" --assume-role-policy-document "$$MAKEFILE_LAMBDA_ASSUME_ROLE_JSON" && \
		echo "* done.")

	@echo "* checking policy..."
	MAKEFILE_POLICY=`aws iam list-policies | jq '.Policies[] | select(.PolicyName == "Lambda$(LAMBDA_FUNCTION)")' -e` && echo "* policy is already created." || \
		MAKEFILE_POLICY=`aws iam create-policy --policy-name "Lambda$(LAMBDA_FUNCTION)" --policy-document "$$MAKEFILE_LAMBDA_POLICY" | jq '.Policy'` &&\
	MAKEFILE_POLICY_ARN=`echo $$MAKEFILE_POLICY | jq '.Arn' -r` && \
	aws iam list-attached-role-policies --role-name "Lambda$(LAMBDA_FUNCTION)" | jq '.AttachedPolicies[] | select(.PolicyName == "Lambda$(LAMBDA_FUNCTION)")' -e && echo "* policy is already attached to role." || \
		(echo "* attach policy to role..." && \
		aws iam attach-role-policy --role-name "Lambda$(LAMBDA_FUNCTION)" --policy-arn "$$MAKEFILE_POLICY_ARN" && \
		echo "* done")

	@echo "* checking lambda function..."
	aws lambda get-function --function-name "$(LAMBDA_FUNCTION)" && echo "* lambda function is already created." || \
		(echo "* lambda function is not created. creating..." && \
		aws lambda create-function --function-name "$(LAMBDA_FUNCTION)" --role "$(shell aws iam get-role --role-name "Lambda$(LAMBDA_FUNCTION)" | jq '.Role.Arn' -r)" --region "$(AWS_REGION)" --package-type "Image" --code "ImageUri=$(AWS_ECR_REPOSITORY_BASE)/$(PACKAGE):$(TAG)" && \
		echo "* done.")

	@echo "* add permission."
	-aws lambda add-permission --function-name arn:aws:lambda:$(AWS_REGION):$(AWS_ACCOUNT):function:$(LAMBDA_FUNCTION) --source-arn arn:aws:execute-api:$(AWS_REGION):$(AWS_ACCOUNT):imj0a2qi30/*/POST/ --principal apigateway.amazonaws.com --statement-id FromAPIGateway --action lambda:InvokeFunction

create-ecr-repository:
ifndef PACKAGE
		@echo variable PACKAGE is not defined.
		@exit 1
endif
	@echo "* cheking ecr repository..."
	aws ecr describe-repositories | jq '.repositories[] | select(.repositoryName == "$(PACKAGE)")' -e && echo "* ecr repository is already created." || \
		(echo "* ecr repository is not created. creating..." && \
		aws ecr create-repository --repository-name "$(PACKAGE)" --image-tag-mutability IMMUTABLE && \
		echo "* done.")



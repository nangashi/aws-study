MAKE=make
PACKAGE=lambda-slack-notification
LAMBDA_FUNCTION=SlackNotification
ARGS=-C ../../ PACKAGE="$(PACKAGE)" LAMBDA_FUNCTION="$(LAMBDA_FUNCTION)"

.PHONY: build
build:
	$(MAKE) $(ARGS) build update-lambda-function

.PHONY: init
init:
	$(MAKE) $(ARGS) create-ecr-repository build create-lambda-function


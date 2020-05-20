# takeon-egest

Variables
Within the AWS provider in aws.tf, we have declared the access_key and secret_access_key as without this, the pipeline gives a credentials error message. The values for these are set in the secrets pipeline.

You will need to change the user in egestParams.tf so that it has your name in it, then it gets called within egest.tf every time ${var.user} is called.

You will also need to amend the lambda function name so as to not overwrite what is already there, if running this for testing purposes

# git versioning

To tag a new release, use the command `git tag -a <tag eg. v1.0> -m <tag message eg. new feature release>`

Then run `git push origin <tag>`

This will trigger the concourse pipeline to build from the latest version
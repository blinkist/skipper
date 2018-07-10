# Skipper

## A tool for working with Amazon ECS clusters.

Skipper is a command line tool that was built to ease working with [Amazon ECS](https://aws.amazon.com/ecs/) clusters.

Skipper offers various subcommands for interacting with ECS: listing tasks, fetching logs, and so on.

### Debugging tasks

One of the most interesting use cases of Skipper is for debugging task instances running on an ECS cluster, using the `skipper shell tunnel` subcommand. 
This command allows a user to safely enter a Docker task by creating an exact copy of the task onto a newly created, isolated cluster that's not serving traffic. 
Skipper then automatically drpos into an interactive session inside the new clone of the task, allowing the user to debug and explore the application environment without affecting the original live instance.

#### How it works

When setting up a cloned debug instance of a task, Skipper creates an EC2 instance with a temporary SSH keypair that it generates for the calling user. 
Skipper then initiates an SSH connection to that instance, and tunnels a Docker client through that connection to connect to the local Docker daemon on the machine. 
This means it can spawn a shell inside the remote Docker container with a similar experience to that of running `docker exec` on the user's local machine.

## Usage

Since Skipper just uses the standard AWS environment variables for authorisation configuration (i.e `AWS_SECRET_KEY` and `AWS_ACCESS_KEY`), it's ideally suited for use in conjunction with [`aws-vault`](https://github.com/99designs/aws-vault):

```
	# list all of the ecs instances in the prod environment
	aws-vault exec dev -- skipper list
	# debug a task in the dev environment
	aws-vault exec prod -- skipper shell tunnel
```

## TODO
- test and document what all subcommands actually do
- remove all exits/panics from non-main packages
- surface all errors currently ignored
- CI (linting, formatting)
- rename `shell tunnel` to `debug`?
- Tests

## Acknowledgements

Skipper is partly inspired by the project [ecsy](https://github.com/lox/ecsy), but is less focused on setting up infrastructure, as the authors believe Terraform is a tool better suited for this.

The authors wish to thank [Blinkist](https://blinkist.com) for their support in the creation of this project.

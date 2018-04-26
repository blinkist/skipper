# Skipper

## A tool for working together with an ECS Cluster.

This project is an Open Source project with the goal in mind to create nice tooling around ECS. It is partly based on the project ECSY but is les focused to setup infrastructure as that Terraform is the tool for this. Many thanks to Blinkist www.blinkist.com 

### The Good

It's a cool tool and it can easily be extended for much more. The cool part of the tool is that it can log a user into a docker task by copying the exact same task on a newly created instance, tunneling the docker socket to the local machine and spawning a shell into that task as if it's on your local machine.

### The Bad
It's far from finished and it started a Golang learning project, Code might not be the cleanest, logical.

### The Ugly
No tests have been written..


## TODO

Docker tunnel:

Better connection handling by using channel tricks.
Clean up instances after use.
Choose shell irb/sh/bash

# Write Tests

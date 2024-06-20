#!/bin/sh

# Set Git config using environment variables
git config --global user.name "$GIT_USER_NAME"
git config --global user.email "$GIT_USER_EMAIL"

# Start the application
./main

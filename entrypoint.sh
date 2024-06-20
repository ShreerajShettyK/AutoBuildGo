#!/bin/sh

# Set Git config using environment variables
git config --global user.name "$GIT_AUTHOR_NAME"
git config --global user.email "$GIT_AUTHOR_EMAIL"

# Start the application
./main

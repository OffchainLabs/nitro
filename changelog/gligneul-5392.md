### Internal
- Only start the redis service container in the CI jobs that use it, avoiding Docker Hub rate limits in the other jobs

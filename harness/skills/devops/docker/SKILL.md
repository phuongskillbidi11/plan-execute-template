---
name: docker
domain: devops
level: technology
description: Docker layering, build-cache invalidation, and the difference between an image problem and a container-runtime problem.
tags: [docker, container, dockerfile, "build cache"]
triggers: [docker, dockerfile, container, "docker compose", "build cache"]
version: "1.0.0"
requires: []
recommends: []
capabilities: []
conflicts: []
when_to_use: Writing/debugging a Dockerfile, a container that behaves differently than the same code run locally, or a slow/broken image build.
when_not_to_use: A pure Kubernetes orchestration question with no Dockerfile/image concern.
---

# Skill: docker

## Purpose

The layering and caching model that explains most "works on my machine, fails in the
container" and "the build is slow" complaints.

## Method

1. **Layer order determines cache invalidation.** Every instruction is a layer; changing a
   line invalidates every layer after it. Put the least-frequently-changing steps
   (installing system/OS packages) before the most-frequently-changing ones (copying
   application source) so an unrelated code change doesn't force a full dependency
   reinstall.
2. **"Works locally, fails in the container" is almost always an environment difference**,
   not a Docker bug — check base image OS/library versions, working directory, environment
   variables, and file permissions (a file copied with `COPY` can end up owned by a
   different user than the one the container runs as) before suspecting Docker itself.
3. **A container's filesystem changes are ephemeral** unless a volume/bind-mount is used —
   data written inside a container without one disappears when the container is removed,
   not just when it stops.
4. **`docker build` context** includes everything not excluded by `.dockerignore` — a
   surprisingly slow build is often caused by an unintentionally large build context (e.g.
   a `node_modules` or `.git` directory being sent to the daemon).

## Anti-patterns

- Debugging a "container-only" failure by staring at the Dockerfile before checking whether
  it's actually an environment/version difference.
- Storing state a service needs to survive a restart with no volume behind it.

# Contributing Guide

This project uses a structured Git workflow.

## Branches

main
Stable version of the project.

dev
Main development branch.

feature/*
Individual feature branches.

Example:

feature/crawler-frontier
feature/html-parser
feature/index-builder

---

## Workflow

1. Start from dev

git checkout dev
git pull origin dev

2. Create a feature branch

git checkout -b feature/<feature-name>

3. Make changes and commit

git add .
git commit -m "component: description"

Example:

crawler: implement frontier queue

4. Push branch

git push origin feature/<feature-name>

5. Open Pull Request

feature/<feature-name> → dev

---

## Rules

Do NOT push directly to main.

All code must go through a Pull Request.

At least one teammate should review before merging.

---

## Commit Messages

Use format:

component: change

Examples:

crawler: add frontier queue
parser: implement HTML link extraction
indexer: add redis storage pipeline

Avoid messages like:

update
fix
changes

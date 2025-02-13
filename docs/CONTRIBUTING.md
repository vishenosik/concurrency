## Common rules

1. It's not allowed to commit into `main` or `dev` branches directly. Commits are meant to be made inside a branch they're attached to.
2. Merge with `main` or `dev` branches are made using merge-requests only.
3. `main` is a release branch.
4. `dev` is a testing branch. `dev` must never be merged with `main`
5. All new branches should be created from `main` branch to prevent merge problems in future.

## Branches

Branches intended for storing solutions to an issue. New branch is created for each issue. The solution branch exists until it is merged with `main` branch. After merging with `main` branch, it is forbidden to make changes to the solution branch. All problems after merging with the current release branch should be fixed in the task tracker and fixed in separate branches.

Branch format: `{TASK_TYPE}/{TASK_TRACKER}-{TASK_NUMBER}-{TASK_SHORT_DESCRIPTION}`

### Branch format parameters
#### TASK_TYPE

A conventional type for a task.

* feat - a new feature
* fix - bugfix/hotfix
* build - changes related to building a project or working with dependencies
* docs - changes related to documentation of a project
* refactor - changes aimed at improving the code, changing the existing feature

#### TASK_NUMBER

Full task number related to task tracker system. It's better to agree on the encodings of some of them:
* YT - YouTrack
* GTH - GitHub
* JR - Jira
* Add new ones after agreement

#### TASK_SHORT_DESCRIPTION

A brief description of the task that reflects the essence of the task, for example, add_something_new

### Examples: 

* feat/GTH-TSK-0001-add_something_new
* fix/YT-TSK-0002-fix_something_broken
* build/JR-TSK-0003-update_go_version
* refactor/GTH-4-rebase_config
* docs/YT-TSK-0005-update_readme

## Commits

Every commit should be written in English. Commit should have all necessary and sufficient information about changes in a code.

Commit format: `{COMMIT_TYPE} ( {SHORT_DESCRIPTION} ): {COMMIT_MESSAGE}`

### Commit format parameters

#### COMMIT_TYPE

* build - changes to dependencies or build scripts
* chore - updating the infrastructure components of the project (clean gitignore, update the Taskfile or Makefile)
* docs - adding feature documentation (code comments, readme update)
* feat - add a new feature
* fix - bugfix/hotfix
* refactor - changes aimed at improving the code, changing the existing feature
* style - Changes that do not affect the operation of the code (formatting, code style, etc.)
* test - writing unit tests, test coverage, etc.
* revert 

#### SHORT_DESCRIPTION

A brief description of the area of the code where the changes occur.

#### COMMIT_MESSAGE

All necessary and sufficient information about changes in a commit.

### Examples: 

* feat(pipeline): Add somethin to config 
* fix(pipeline): Fix panic, that occurs in Run() func.
* docs(pipeline): Update func comments.
* test(pipeline): Add new fail test.

[Return to Readme](../README.md)
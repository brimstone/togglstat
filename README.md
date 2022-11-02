togglstat
=========

Simple tool for converting time tracked in toggl to a timecard like format.


## Setup

Run once with your api token:
```
togglstat -token aabbccddee...
```

Edit `$HOME/.togglstat.yaml` as needed:

| Value            | Description                                                                                           |
| ---------------- | ----------------------------------------------------------------------------------------------------- |
| `apitoken`       | API Token for toggl                                                                                   |
| `projects`       | List of projects automatically gathered from toggl's API. No need to change.                          |
| `clients`        | List of clients automatically gathered from toggl's API. No need to change.                           |
| `skipprojects`   | List of project names to never consider.                                                              |
| `renameprojects` | Dictionary of projects to rename. Useful for subdividing tasks in toggl, but combining for reporting. |


## Usage

Basic usage is without arguments:
```
togglstat
```

Automagic upgrade available with the `upgrade` subcommand:
```
togglstat upgrade
```

Pairs well with [xbar](https://xbarapp.com/)

# git-chat

A chat app where **GitHub** is the **Backend**. Messages are a simple text files in a private repository, sent by committing and pushing, read by pulling and parsing filenames. No servers, no databases, no signups. Just git and GitHub collaborators.

![Go](https://img.shields.io/badge/Go-00ADD8?style=flat-square&logo=go&logoColor=white)
![Cobra](https://img.shields.io/badge/Cobra-000000?style=flat-square&logo=go&logoColor=white)
![Bubble Tea](https://img.shields.io/badge/Bubble_Tea-FF69B4?style=flat-square&logo=go&logoColor=white)

## Prerequisites

- [GitHub CLI (`gh`)](https://cli.github.com/) installed and authenticated
- GitHub token scopes: `repo` (for everything) and `delete_repo` (for deleting groups)

## Install

```bash
git clone https://github.com/1hemmem/git-chat.git
cd git-chat
make install
```

Builds the binary and installs it to `~/.local/bin`. Make sure `~/.local/bin` is on your `$PATH`.

Other useful targets: `make help`, `make clean`.

## Get started

### 1. Set up your auth scopes

The `repo` scope is required for almost everything. Run this once to make sure you have it:

```bash
git-chat auth refresh repo delete_repo
```

This opens your browser. Follow the GitHub prompt to grant the scopes.

### 2. Create a group

A "group" is just a private GitHub repository created in your account:

```bash
git-chat creategroup mygroup
```

This creates a private repo tagged `chat-over-git-repo`, clones it locally, writes a README with setup instructions, and pushes an initial commit. The tag is how git-chat discovers your groups later, `listgroups`, `sendmsg`, `readmsgs`, and `open` all search for repos with this tag.

### 3. Add people to the group

Add GitHub users as collaborators on the repo:

```bash
git-chat addmember mygroup 3abde9a
git-chat addmember mygroup moha
```

They have to accept the invite in github so they can join the chat.

### 4. Send messages

Send a message and git-chat writes it as a `.txt` file, commits, and pushes:

```bash
git-chat sendmsg mygroup "Hey everyone, how's it going?"
```

If you don't have write access to the repo, you'll get a "push denied" error.

### 5. Read messages

Pull the latest messages and display them in chronological order:

```bash
git-chat readmsgs mygroup
```

Output looks like this:

```
[2026-06-17 12:00] 3abde9a: Hey everyone, how's it going?
[2026-06-17 12:01] moha: Doing great!
```

### 6. Open the live chat TUI

For a real-time-ish chat experience, launch the terminal UI:

```bash
git-chat open mygroup
```

The TUI polls for new messages every half-second, shows them with colored authors, and lets you send messages inline. Press `Enter` to send, `Esc` to quit.

### 7. List all your groups

```bash
git-chat listgroups
```

This searches GitHub for all repos tagged `chat-over-git-repo` that you have access to (limited to 100 results).

### 8. Delete a group

```bash
git-chat deletegroup mygroup
```

Requires the `delete_repo` scope. It will prompt for confirmation before deleting the entire repo.

---

## All commands

| Command                              | What it does                              |
| ------------------------------------ | ----------------------------------------- |
| `git-chat auth refresh [scopes]`     | Refresh GitHub auth with the given scopes |
| `git-chat creategroup <name>`        | Create a private repo, clone, seed README |
| `git-chat addmember <group> <user>`  | Add a GitHub user as collaborator         |
| `git-chat sendmsg <group> <message>` | Send a message to a group                 |
| `git-chat readmsgs <group>`          | Read all messages in a group              |
| `git-chat open <group>`              | Open the live-chat TUI                    |
| `git-chat listgroups`                | List groups you have access to            |
| `git-chat deletegroup <group>`       | Delete a group (and its repo)             |

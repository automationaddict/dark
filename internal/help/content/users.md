# Users

Browse the local user accounts on this machine: regular users (UID ≥ 1000) plus root. See who has sudo rights, who's currently logged in via which TTY or seat, password-aging details from `/etc/shadow`, and which processes each user owns. All write operations — adding users, removing users, changing shells, password changes, group membership — go through `dark-helper` + pkexec because they touch `/etc/passwd`, `/etc/shadow`, and `/etc/group`.

Press `?` at any time to open this help. Press `esc` to close it.

## What you see when you land here

The Users page has three sub-sections in an inner sidebar:

1. **Identity** — username, UID, GID, full name (from GECOS), home directory, shell, primary group, group memberships, admin (wheel) flag
2. **Account** — password set / locked / empty, password age, last changed, min/max/warn/inactive day settings from `/etc/shadow`, expiration date
3. **Security** — active sessions (from `loginctl`), recent logins (from `last`), total login count, process count for this user

The user list is in a left-side table; the three sub-sections on the right change to show different views of the currently highlighted user.

## How dark reads users

- **`/etc/passwd`** is parsed directly for the user list. Dark filters out system accounts by default — only UID 0 (root) and UID ≥ 1000 are shown, with UID 65534 (`nobody`) explicitly excluded.
- **`/etc/group`** is parsed for the group name table and for the member-of lookup (which groups each user belongs to).
- **`/etc/shadow`** is the privileged bit. Dark tries to read it directly first (only works if dark is running as root, which it isn't normally). If that fails, dark calls `dark-helper read-shadow` via pkexec — the one-time prompt unlocks the shadow-backed fields like password age and lock state. Decline the prompt and those rows stay blank; everything else still works.
- **`loginctl list-sessions`** and **`last -n 5 -F <user>`** provide the session and login history rows. These need read permission on `/var/log/wtmp`, which is usually world-readable, so no pkexec prompt.
- **`/proc/*/status`** is walked to count processes per UID.

## Navigating — focus and the key flow

1. On landing, focus is on the user list on the left. `j`/`k` moves between users and the right-hand panel updates to show that user's details.
2. Press `enter` to interact with whichever sub-section is visible (identity / account / security).
3. Action keys apply per-user changes — the selected user is always the target.
4. `esc` backs out to the Settings sidebar.

## Actions — the complete keybinding reference

### User management

- `a` — open the Add user dialog. Four fields: Username, Full name (GECOS), Shell (defaults to `/bin/bash`), Admin (y/n for wheel membership).
- `d` — open the Remove user dialog. Confirmation-only. Toggle `--remove-home` in the dialog to also delete the user's home directory. Without that flag, only `/etc/passwd` and `/etc/group` are updated.
- `s` — open the Change shell dialog. Dark reads `/etc/shells` and offers a select list of installed shells. Defaults to the user's current shell.
- `c` — open the Rename (full name / GECOS) dialog. Edits just the comment field, doesn't touch the username.
- `w` — toggle admin flag (wheel group membership). Runs `gpasswd -a <user> wheel` or `gpasswd -d <user> wheel`.

### Account security

- `l` — toggle the account lock state. Runs `usermod -L` (lock) or `usermod -U` (unlock). Locked accounts can't log in via password but ssh key-based auth may still work depending on your sshd_config.
- `p` — change password. Two-step dialog: verify current password (if the target user is yourself), then enter the new password twice. The new password goes through `chpasswd` via `dark-helper`.

### Group membership

- `g` — open the Add group membership dialog. Type a group name; dark runs `gpasswd -a <user> <group>` on commit.
- `G` — open the Remove group membership dialog. Type the group name to remove the user from.

### Universal

- `j` / `k` or `↑` / `↓` — move the user selection
- `enter` — drill into the selected row (where applicable)
- `esc` — back out
- `?` — open this help drawer

## Dialogs

### Add user

Four fields validated before submit:

- **Username** — must match `^[a-z_][a-z0-9_-]*$` (standard Linux username rules) and be ≤ 32 characters. Reserved names (`root`, `nobody`, `daemon`, `bin`, `sys`) are rejected.
- **Full name** — free text, written to the GECOS field
- **Shell** — must be an absolute path that exists on disk. Dark pre-fills from `/etc/shells` but you can type anything valid.
- **Admin** — if checked, the new user is added to the `wheel` group after `useradd` completes, giving them sudo access.

On submit, dark runs `useradd -m [-c <comment>] [-s <shell>] <username>` via dark-helper, then optionally `gpasswd -a <username> wheel`.

### Remove user

Confirmation dialog. The optional `--remove-home` flag controls whether `userdel -r` is used (delete home directory and mail spool) or plain `userdel` (just the account).

### Change password

The flow varies based on whether you're changing your own password or another user's:

- **Your own**: dark asks for the current password first, verifies it with `su -c true <username>`, then prompts for the new password twice.
- **Another user's**: if you have sudo access, dark only asks for the new password. The verification step is skipped.

New passwords are piped into `chpasswd` via dark-helper — never written to disk, never logged, never passed on the command line.

## Common tasks

### Add a new user with sudo rights

1. Press `a`.
2. Username: `alice`. Full name: `Alice Smith`. Shell: `/bin/bash`. Admin: yes.
3. `enter`. The dialog closes and the user list refreshes to show the new entry.
4. The user won't have a password yet — they can't log in until you set one. Select the row and press `p` to open the password dialog.

### Remove a user but keep their home directory

1. Highlight the user. Press `d`.
2. Confirm the dialog **without** checking `--remove-home`.
3. The account is deleted from `/etc/passwd` and `/etc/shadow`, but `/home/<user>/` stays intact — another admin could later reinstate the account by recreating the user with the same UID.

### Change your own shell to fish

1. Highlight your row. Press `s`.
2. The dialog pre-fills with your current shell and offers a select list.
3. Arrow to `/usr/bin/fish` (or type it if not in the select list) and `enter`.
4. New logins use fish. The current session keeps its old shell until you log out.

### Lock out a user without deleting them

1. Highlight the row. Press `l`.
2. The Account row flips to show `locked`. The user can no longer log in via password.
3. Press `l` again to unlock.

### Give a user access to the docker group

1. Highlight the user. Press `g`.
2. Type `docker`. `enter`.
3. The Groups row updates to include `docker`. The user needs to log out and back in for the group membership to take effect in their session.

### See who's logged in right now

1. Navigate to the Security sub-section.
2. Highlight the user.
3. The Sessions list shows every active `loginctl` session for that user: TTY, remote host (if SSH), time since login, session state.

### Change password aging rules

Password aging (max days, min days, warn days, inactive days) isn't directly editable from dark yet — the Account sub-section shows the values read from `/etc/shadow` but doesn't have an edit flow. Use `chage` from a terminal for now.

## Data sources, for the curious

- **`/etc/passwd`** — user list (parsed directly, no helper needed)
- **`/etc/group`** — group name table and member-of lookup
- **`/etc/shadow`** — password state (read via `dark-helper read-shadow` when dark can't read it directly)
- **`loginctl list-sessions`** and **`loginctl show-session`** — active sessions
- **`last -n 5 -F <user>`** — last-login history
- **`/proc/*/status`** — process counts per UID
- **`/etc/shells`** — valid shell list for the Change Shell dialog

Every write goes through `dark-helper` via pkexec — dark never calls `useradd`, `userdel`, `usermod`, `gpasswd`, or `chpasswd` directly.

## Known limitations

- Password aging (chage-style settings) is read but not editable.
- Adding a user to multiple groups at create time requires one `g` press per group after creation. There's no "add to these groups" field in the Add dialog.
- Dark doesn't manage SSH authorized_keys — that's a per-user config file dark doesn't touch.
- The Security sub-section's login history is capped at 5 entries. For more history use `last` directly.
- Changing your own password while logged in over SSH is risky if the change fails — dark uses chpasswd which is atomic, but the session verification (`su -c true`) may fail in restricted shells.
- Group creation (`groupadd`) isn't wrapped. You can add users to existing groups but can't create new ones from dark.

# Git Chat

An over-engineered, decentralized, and strictly immutable chat application where **every single message is a Git commit**.

Traditional chat applications use centralized databases where messages can be deleted or altered by server administrators. Git Chat turns this concept upside down by using Git's cryptographic hash chain (Merkle tree) as a tamper-proof database. If someone attempts to delete or modify a past message, the cryptographic hash chain breaks, creating a divergent timeline and preventing synchronization with other peers.

There are no take-backs here. What is committed, is committed forever.

---

## How It Works

* **Go-Git Core (`go-git/v6`):** Programmatically creates and manages a local Git repository (`./repoDB`). Chat messages are stored as empty commits (`--allow-empty`) signed with the user's Git name and email.
* **Event-Driven Sync (`fsnotify`):** Instead of polling the Git history with expensive CPU loops, the backend monitors the `.git/refs/heads` directory for filesystem events (`write`, `create`, `rename`). When a branch reference updates, new commits are detected and processed instantly.
* **Real-Time WebSocket Hub:** Uses `gorilla/websocket` with a Hub/Client architecture and ping/pong keepalives to broadcast incoming commits immediately to all active browser sessions.
* **Paginated Message History:** Exposes a paginated REST endpoint (`GET /history?page=N`) fetching 50 commits per page in reverse chronological order, supporting smooth infinite scrolling in the frontend while preventing high RAM usage.
* **Decentralized Synchronization:** Automatically pushes new messages to a remote Git repository (`origin`) and runs a background synchronization loop every 5 seconds to pull updates from other peers.

---

## Architecture & Data Flow

```text
[ Browser Client A ]
       │  POST /message
       ▼
[ Gin Web Server ] ──(Commit)──► [ Local repoDB ] ──(Push)──► [ Remote Bare Repo (origin) ]
       ▲                                │                                  │
       │                           (fsnotify)                         (Pull 5s)
       │                                │                                  ▼
[ WebSocket Hub ] ◄────(New Commits)────┘                        [ Client B repoDB ]
       │                                                                   │
       ▼                                                              (fsnotify)
[ Connected Browsers ]                                                     ▼
                                                                 [ Client B WebSocket ]
```

---

## Requirements & Prerequisites

To run this project, you need:

1. **[Go](https://go.dev):** Version `1.27+` (or compatible Go release).
2. **Git CLI:** Installed and accessible in your system's `PATH`.
3. **Git User Identity (CRITICAL):**
   The application automatically reads your Git configuration via `git config user.name` and `git config user.email` to sign chat messages.

   > ⚠️ **Important:** If `user.name` or `user.email` is not configured, the application will fail to start on launch (`panic: user.name is empty`).

   Configure your identity globally before running the server:
   ```bash
   git config --global user.name "Your Name"
   git config --global user.email "your.email@example.com"
   ```
4. **Modern Web Browser:** Compatible with HTML5, Fetch API, and WebSockets.

---

## Getting Started

Follow these steps to set up and run Git Chat locally:

### 1. Clone the Repository
```bash
git clone https://github.com/Xelckis/gitchat.git
cd gitchat
```

### 2. Set Up the "Database" (`repoDB`)
The application stores all message commits in a local folder named `repoDB` located at the project root.

Initialize the repository and create an initial commit:
```bash
mkdir repoDB
cd repoDB
git init
git commit --allow-empty -m "Initial commit"
cd ..
```

> **Why is the initial commit required?**
> The filesystem watcher (`fsnotify`) and Git head resolver (`r.Head()`) require the default branch reference file (e.g., `.git/refs/heads/main` or `master`) to exist before starting the server.

### 3. Run the Backend Server
Install dependencies and launch the Gin server:
```bash
go mod tidy
go run main.go
```

The server will start listening on port `8081` (default: `http://localhost:8081`).

### 4. Start Chatting
Open your browser and navigate to:
```
http://localhost:8081
```

---

## API Reference

### HTTP Endpoints

| Method | Endpoint | Description | Query / Body Parameters | Response |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/` | Serves the single-page chat interface (`index.html`) | None | `text/html` |
| `POST` | `/message` | Submits a new chat message | Form field `message` (`string`, required) | HTTP 200 (or JSON error) |
| `GET` | `/history` | Fetches paginated past messages (50 items/page) | `page` (optional `int`, default: `1`) | JSON array of `Commit` objects |

#### Commit JSON Schema Example:
```json
[
  {
    "Hash": "e2a1b94c30c89df7123456789abcdef012345678",
    "AuthorName": "Alice",
    "AuthorEmail": "alice@example.com",
    "When": "2026-09-16T10:30:00Z",
    "Committer": "Alice <alice@example.com>",
    "Message": "Hello from Git Chat!"
  }
]
```

### WebSocket Endpoint

| Endpoint | Protocol | Description |
| :--- | :--- | :--- |
| `/ws` | `ws://` / `wss://` | Real-time bi-directional connection. Streams new `Commit` objects as JSON whenever commits are detected in `repoDB`. |

---

## Multiplayer Mode (Decentralized Synchronization)

To chat with other users across different machines or networks, synchronize your local `repoDB` instances with a shared Bare Git repository:

### 1. Set Up a Central Bare Repository
Create a bare repository on a VPS, local server, or Git host:
```bash
git init --bare /path/to/chat.git
```

### 2. Connect Your Local `repoDB` to the Remote
Inside each participant's `repoDB` folder, add the remote origin and push the initial commit:
```bash
cd repoDB
git remote add origin <url-or-path-to-bare-repo>
git branch -M main
git push -u origin main
cd ..
```

### 3. Automatic Synchronization in Action
* **Sending Messages:** When a user posts a message, the server creates a local commit in `repoDB` and immediately pushes it to `origin`.
* **Receiving Messages:** Each peer's backend runs `StartSyncLoop`, which executes `git pull origin` every 5 seconds.
* **Instant Delivery:** When pulled commits update `.git/refs/heads/main`, `fsnotify` detects the change and pushes the new messages directly to connected browsers via WebSocket without needing to refresh the page.

---

## Project Structure

```text
gitchat/
├── index.html              # Frontend UI (HTML5, CSS3, Vanilla JS, WebSockets)
├── main.go                 # Main entrypoint, HTTP router configuration & lifecycle
├── internal/
│   ├── gitLogic/
│   │   ├── git.go          # Git repository operations (commit, pull, push, log history)
│   │   └── watcher.go      # fsnotify file watcher on .git/refs/heads
│   └── web/
│       ├── handlers.go     # Gin HTTP handlers (/message, /history)
│       └── websocket.go    # Gorilla WebSocket Hub and Client connection management
├── repoDB/                 # (Created at setup) Local Git database repository
├── go.mod                  # Go module definition and dependencies
├── go.sum                  # Dependency lockfile
└── README.md               # Project documentation
```

---

## Troubleshooting & FAQ

#### `panic: user.name is empty` or `panic: user.email is empty`
* **Cause:** Git user configuration is missing on the machine.
* **Fix:** Run:
  ```bash
  git config --global user.name "Your Name"
  git config --global user.email "your.email@example.com"
  ```

#### `Failed to start Git monitoring: ...`
* **Cause:** The `repoDB` directory does not exist or has not had an initial commit created.
* **Fix:** Ensure step 2 of *Getting Started* was executed (`mkdir repoDB && cd repoDB && git init && git commit --allow-empty -m "Initial commit"`).

#### `warning: failed sending message to remote repo`
* **Cause:** No remote `origin` is configured in `repoDB` (or the remote is unreachable).
* **Fix:** This warning is expected and harmless when running locally in single-player mode. If using multiplayer mode, verify `git remote -v` inside `repoDB`.

#### Port `8081` is already in use
* **Fix:** Change the port passed to `router.Run(":8081")` in `main.go` to any free port (e.g., `:8082`).

---

## License

This project is open-source and available under the [Apache License 2.0](LICENSE).
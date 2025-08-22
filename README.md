# TaskMaster Weekly Planner 📅

[![Go Version](https://img.shields.io/github/go-mod/go-version/sedobrengocce/TaskMaster)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Docker](https://img.shields.io/badge/Docker-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)

A **self-hosted** web platform for managing personal and collaborative tasks, with a core focus on weekly planning. The project is designed to give the user full control and ownership of their data.

## ✨ Key Features

* **Intuitive Weekly Planning**: A dashboard interface focused on the current week (Monday-Sunday).
* **Flexible Task Management**: Support for both single-use ("disposable") tasks and reusable templates for recurring tasks.
* **Project-Based Organization**: Group tasks into themed projects, each with a customizable color.
* **Simple Collaboration**: Share individual tasks or entire projects with other registered users.
* **Secure Logout**: JWTs are invalidated upon logout using a Redis-based denylist.
* **Self-Hosted with Docker**: Simple and consistent installation and updates on any operating system.
* **RESTful API**: A robust backend that exposes all functionality via an API, ready for future extensions.

---

## 🛠️ Tech Stack

| Component | Technology |
| :--- | :--- |
| **Backend** | [**Go**](https://golang.org/) with the [**Echo**](https://echo.labstack.com/) framework |
| **Database** | [**MariaDB**](https://mariadb.org/) |
| **Cache / Denylist** | [**Redis**](https://redis.io/) |
| **DB Access** | [**sqlc**](https://sqlc.dev/) (type-safe code generation from SQL queries) |
| **Frontend** | [**Templ**](https://templ.guide/) (templating) + [**HTMX**](https://htmx.org/) (interactivity) |
| **Build Automation**| [**Make**](https://www.gnu.org/software/make/) |
| **Containerization**| [**Docker**](https://www.docker.com/) & [**Docker Compose**](https://docs.docker.com/compose/) |
| **VCS** | [**Jujutsu (jj)**](https://github.com/martinvonz/jj) with a [**Git**](https://git-scm.com/) backend |
| **Migrations**| [**golang-migrate**](https://github.com/golang-migrate/migrate) |
| **Live Reload**| [**Air**](https://github.com/cosmtrek/air) |

---

## 🚀 Getting Started

Follow these steps to get the project running locally.

### Prerequisites

Ensure you have the following tools installed on your local machine:

1.  **Docker** & **Docker Compose**
2.  **Make**

The Go toolchain, `golang-migrate`, and `air` are all managed within the Docker environment, so you don't need to install them locally.

### Installation & Setup

1.  **Clone the Repository**
    ```bash
    git clone [https://github.com/sedobrengocce/TaskMaster.git](https://github.com/sedobrengocce/TaskMaster.git)
    cd TaskMaster
    ```

2.  **Configure Environment Variables**
    Create a `.env` file by copying the provided sample file. **It's crucial to review and change the secret keys for a secure setup.**

    ```bash
    cp .env.sample .env
    ```
    Your `.env` file will look like this:
    ```env
    # --- Database Credentials ---
    DB_NAME=weekly_platform
    DB_USER=platform_user
    DB_PASSWORD=una_password_molto_sicura
    DB_ROOT_PASSWORD=una_password_di_root_ancora_piu_sicura

    # --- Redis Credentials ---
    REDIS_PASSWORD=una_password_per_redis_molto_sicura

    # --- Secrets ---
    JWT_SECRET=una_secret_key_molto_sicura
    REFRESH_SECRET=altra_secret_key_molto_sicura
    ```

3.  **Launch the Development Environment**
    Run the following command to build the containers, start the services, and apply the initial database migrations automatically:

    ```bash
    make -f Makefile.dev dev
    ```
    The application will start with live-reloading. Any changes to `.go` files will automatically restart the server.

---

## ⚙️ Usage

This project uses a `Makefile.dev` to simplify managing the Docker environment.

### Running in Development Mode
This is the standard command for daily development. It starts all services and enables live-reloading.

```bash
make -f Makefile.dev dev
```

### Running in Production Mode
This command builds and starts the containers using the optimized production image.

```bash
make -f Makefile.dev prod
```

### Stopping the Environment
To stop all running containers for the project, use:
```bash
make -f Makefile.dev down
```

### Accessing the Application

Once running, the application will be accessible at:
`http://localhost:3000`

**Key API Endpoints:**
* `GET /healthcheck`: Checks the server's status.
* `POST /api/register`: Creates a new user.
* `POST /api/login`: Authenticates a user and returns a JWT.
* `POST /api/logout`: Logs the user out and invalidates the current token.

---

## 🌿 Version Control

This project uses **Jujutsu (jj)** as the primary version control interface, relying on a Git backend for compatibility with GitHub.

## 📜 License

Distributed under the MIT License.

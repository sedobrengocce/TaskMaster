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
* **Self-Hosted with Docker**: Simple and consistent installation and updates on any operating system.
* **RESTful API**: A robust backend that exposes all functionality via an API, ready for future extensions (e.g., a mobile app).

---

## 🛠️ Tech Stack

| Component          | Technology                                                                                                        |
| :----------------- | :---------------------------------------------------------------------------------------------------------------- |
| **Backend** | [**Go**](https://golang.org/) with the [**Echo**](https://echo.labstack.com/) framework                             |
| **Database** | [**MariaDB**](https://mariadb.org/)                                                                               |
| **DB Access** | [**sqlc**](https://sqlc.dev/) (type-safe code generation from SQL queries)                                      |
| **Frontend** | [**Templ**](https://templ.guide/) (templating) + [**HTMX**](https://htmx.org/) (interactivity)                      |
| **Styling** | [**SCSS**](https://sass-lang.com/)                                                                                |
| **Containerization**| [**Docker**](https://www.docker.com/) & [**Docker Compose**](https://docs.docker.com/compose/)                    |
| **VCS** | [**Jujutsu (jj)**](https://github.com/martinvonz/jj) with a [**Git**](https://git-scm.com/) backend                   |
| **Migrations** | [**golang-migrate**](https://github.com/golang-migrate/migrate)                                                   |
| **Live Reload** | [**Air**](https://github.com/cosmtrek/air)                                                                        |

---

## 🚀 Getting Started

Follow these steps to get the project running locally.

### Prerequisites

Ensure you have the following tools installed:

1.  **Docker** & **Docker Compose**
2.  **golang-migrate CLI** - [Installation Instructions](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate#installation)

### Installation & Setup

1.  **Clone the Repository**
    ```bash
    git clone [https://github.com/sedobrengocce/TaskMaster.git](https://github.com/sedobrengocce/TaskMaster.git)
    cd TaskMaster
    ```

2.  **Configure Environment Variables**
    Create a `.env` file in the project root by copying the `env.example` template (if it exists) or by using the content below. **Replace the placeholder values with your own secure credentials.**

    ```env
    # --- Database Credentials ---
    DB_NAME=weekly_platform
    DB_USER=platform_user
    DB_PASSWORD=a_very_secure_password
    DB_ROOT_PASSWORD=an_even_more_secure_root_password
    ```

3.  **Start the Database**
    Start only the database container to be able to run migrations.
    ```bash
    docker-compose up -d db
    ```

4.  **Run Database Migrations**
    Apply the database schema using the `golang-migrate` CLI. Make sure the database is fully up and running before executing this command.

    ```bash
    migrate -database 'mysql://platform_user:a_very_secure_password@tcp(localhost:3306)/weekly_platform' -path db/migrations up
    ```

---

## ⚙️ Usage

### Running in Development Mode

With Development mode, the application will start with **live-reloading**. Any changes to `.go` files will automatically restart the server.

```bash
make -f Makefile.dev dev
```

### Running in Production Mode

To simulate a production environment, run this command. This will build a minimal, optimized Docker image.

```bash
make -f Makefile.dev prod
```

### Accessing the Application

Once running, the application will be accessible at:
`http://localhost:3000`

**Key Endpoints:**
* `GET /healthcheck`: Checks the server's status.
* `POST /api/register`: Creates a new user.

---

## 🌿 Version Control

This project uses **Jujutsu (jj)** as the primary version control interface, relying on a Git backend for compatibility with GitHub. Contributors should familiarize themselves with the `jj` workflow.

## 📜 License

Distributed under the MIT License. See `LICENSE` for more information.

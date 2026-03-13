# 01 - Task CRUD Handlers e Route

## Descrizione
Implementare gli handler HTTP e le route per le operazioni CRUD sui task. Le query SQLC esistono gia in `internal/db/queries/tasks.sql` e il codice generato e in `internal/db/tasks.sql.go`. Mancano gli handler in `internal/server/handlers.go` e le route in `internal/server/routes.go`.

## Sotto-task

- [ ] Implementare `CreateTaskHandler` — validare input (title obbligatorio, project_id opzionale, task_type enum, priority), chiamare `db.CreateTask`
- [ ] Implementare `ListTasksByProjectHandler` — ricevere `project_id` da path param, chiamare `db.GetTaskListByProjectId`
- [ ] Implementare `ListTasksByUserHandler` — estrarre user ID dal JWT, chiamare `db.GetTasksByUserId` (include task condivisi via LEFT JOIN)
- [ ] Implementare `UpdateTaskHandler` — validare input parziale, chiamare `db.UpdateTask`
- [ ] Implementare `DeleteTaskHandler` — chiamare `db.DeleteTask`
- [ ] Registrare le route in `routes.go`:
  - `POST /api/tasks`
  - `GET /api/tasks` (by user)
  - `GET /api/projects/:id/tasks` (by project)
  - `PUT /api/tasks/:id`
  - `DELETE /api/tasks/:id`
- [ ] Scrivere test unitari per ogni handler (seguire il pattern esistente con mock querier)

## File coinvolti

- `internal/server/handlers.go` — aggiungere handler
- `internal/server/routes.go` — registrare route
- `internal/server/handlers_test.go` — test
- `internal/db/queries/tasks.sql` — query gia presenti
- `internal/db/tasks.sql.go` — codice generato gia presente

## Dipendenze

- Nessuna — le query DB e i model sono gia generati

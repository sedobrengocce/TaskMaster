# 04 - Task Sharing (Handler + Route)

## Descrizione
Implementare gli handler per la condivisione dei singoli task. Le query SQLC `ShareTaskWithUser` e `UnshareTaskWithUser` esistono gia in `tasks.sql`. La tabella `shared_tasks` e definita nella migrazione `000005`. Servono handler e route analoghi a quelli dei progetti.

## Sotto-task

- [ ] Implementare `ShareTaskHandler` — riceve `task_id` da path e `user_id` dal body, chiama `db.ShareTaskWithUser`
- [ ] Implementare `UnshareTaskHandler` — riceve `task_id` da path e `user_id` dal body, chiama `db.UnshareTaskWithUser`
- [ ] Registrare le route in `routes.go`:
  - `POST /api/tasks/:id/share`
  - `DELETE /api/tasks/:id/share`
- [ ] Gestire errori: task non trovato, utente non trovato, gia condiviso (UNIQUE constraint)
- [ ] Scrivere test unitari
- [ ] Aggiornare mock se necessario

## File coinvolti

- `internal/server/handlers.go` — nuovi handler
- `internal/server/routes.go` — nuove route
- `internal/server/handlers_test.go` — test
- `internal/db/queries/tasks.sql` — query gia presenti

## Dipendenze

- `01-task-crud-handlers.md` — servono le route base dei task
- `03-wire-sharing-routes.md` — seguire lo stesso pattern dei progetti

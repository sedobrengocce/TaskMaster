# 05 - Controlli di Ownership e Autorizzazione

## Descrizione
Attualmente gli handler non verificano che l'utente autenticato sia il proprietario della risorsa o abbia accesso tramite sharing. Un utente potrebbe modificare/eliminare progetti e task di altri. Servono controlli di autorizzazione su tutte le operazioni di modifica.

## Sotto-task

- [ ] Creare helper `isProjectOwnerOrShared(ctx, projectID, userID)` che verifica ownership o accesso condiviso
- [ ] Creare helper `isTaskOwnerOrShared(ctx, taskID, userID)` analogo per i task
- [ ] Aggiungere check di ownership su:
  - `UpdateProject` — solo il proprietario puo modificare
  - `DeleteProject` — solo il proprietario puo eliminare
  - `SharedProjectHandler` — solo il proprietario puo condividere
  - `UnshareProjectHandler` — solo il proprietario puo revocare
- [ ] Aggiungere check di ownership su:
  - `UpdateTaskHandler` — solo il creatore puo modificare
  - `DeleteTaskHandler` — solo il creatore puo eliminare
  - `ShareTaskHandler` — solo il creatore puo condividere
  - `UnshareTaskHandler` — solo il creatore puo revocare
- [ ] Aggiungere check di accesso (owner OR shared) su:
  - `GetProject` — proprietario o utente con sharing
  - `ListTasksByProjectHandler` — proprietario del progetto o con sharing
  - `CompleteTaskHandler` — proprietario o utente con sharing
- [ ] Aggiungere query SQLC se mancanti:
  - `IsProjectSharedWithUser` — SELECT EXISTS
  - `IsTaskSharedWithUser` — SELECT EXISTS
- [ ] Ritornare `403 Forbidden` per accesso non autorizzato (distinto da `401 Unauthorized`)
- [ ] Scrivere test per ogni scenario di autorizzazione
- [ ] Rigenerare SQLC se aggiunte nuove query

## File coinvolti

- `internal/server/handlers.go` — aggiungere check in ogni handler
- `internal/server/middleware.go` — eventuale middleware di autorizzazione
- `internal/db/queries/projects.sql` — query di verifica ownership
- `internal/db/queries/tasks.sql` — query di verifica ownership
- `internal/server/handlers_test.go` — test di autorizzazione

## Dipendenze

- `01-task-crud-handlers.md` — servono gli handler dei task
- `03-wire-sharing-routes.md` — servono le route di sharing progetti
- `04-task-sharing.md` — servono le route di sharing task

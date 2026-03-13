# 02 - Task Completion e Logging

## Descrizione
Implementare il sistema di completamento task tramite la tabella `task_logs` (migrazione `000002`). Un task viene "completato" inserendo un record in `task_logs` con `completed_by_user_id` e `completed_at`. Per i task ripetitivi, ogni completamento e un log separato; per i task singoli, il completamento e finale.

## Sotto-task

- [ ] Aggiungere query SQLC in `internal/db/queries/tasks.sql`:
  - `CompleteTask` — INSERT in `task_logs`
  - `UncompleteTask` — DELETE da `task_logs` (per il giorno corrente o per ID)
  - `GetTaskCompletions` — SELECT da `task_logs` per un dato `task_id` (con range di date opzionale)
  - `GetCompletionsForWeek` — SELECT completamenti per user in un range settimanale
- [ ] Rigenerare codice SQLC: `sqlc generate`
- [ ] Implementare `CompleteTaskHandler` — POST `/api/tasks/:id/complete`
- [ ] Implementare `UncompleteTaskHandler` — DELETE `/api/tasks/:id/complete`
- [ ] Implementare `GetTaskCompletionsHandler` — GET `/api/tasks/:id/completions`
- [ ] Registrare le route in `routes.go`
- [ ] Scrivere test unitari
- [ ] Aggiornare `mock_querier.go` con i nuovi metodi

## File coinvolti

- `internal/db/queries/tasks.sql` — nuove query
- `internal/db/*.sql.go` — rigenerati da SQLC
- `internal/db/mock_querier.go` — aggiornare mock
- `internal/server/handlers.go` — nuovi handler
- `internal/server/routes.go` — nuove route
- `internal/server/handlers_test.go` — test

## Dipendenze

- `01-task-crud-handlers.md` — servono le route base dei task

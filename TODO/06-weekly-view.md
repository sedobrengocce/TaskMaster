# 06 - Filtro Settimanale sui Task

## Descrizione
Funzionalita core di TaskMaster: visualizzare i task e i relativi completamenti per una settimana specifica (lunedi-domenica). L'utente deve poter vedere quali task ha completato in una data settimana e quali sono ancora da fare.

## Sotto-task

- [ ] Aggiungere query SQLC `GetWeeklyTaskView`:
  - Parametri: `user_id`, `week_start` (DATE, lunedi), `week_end` (DATE, domenica)
  - Ritorna task dell'utente con flag di completamento per ogni giorno della settimana
  - Includere task condivisi
- [ ] Aggiungere query `GetCompletionsInRange`:
  - Parametri: `user_id`, `start_date`, `end_date`
  - Ritorna tutti i completamenti nel range
- [ ] Rigenerare SQLC
- [ ] Implementare `WeeklyViewHandler`:
  - `GET /api/weekly?week=2026-03-09` (il lunedi della settimana)
  - Se `week` non specificato, usare la settimana corrente
  - Calcolare automaticamente lunedi-domenica
  - Ritornare task + completamenti aggregati per giorno
- [ ] Definire struct di risposta JSON con struttura settimanale
- [ ] Registrare la route
- [ ] Scrivere test

## File coinvolti

- `internal/db/queries/tasks.sql` — nuove query
- `internal/db/*.sql.go` — rigenerati
- `internal/server/handlers.go` — nuovo handler
- `internal/server/routes.go` — nuova route
- `internal/server/handlers_test.go` — test

## Dipendenze

- `01-task-crud-handlers.md` — servono i task
- `02-task-completion-logging.md` — serve il sistema di completamento

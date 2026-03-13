# 03 - Collegare le Route di Sharing Progetti

## Descrizione
Gli handler `SharedProjectHandler` e `UnshareProjectHandler` esistono gia in `handlers.go` (righe ~516-562), ma le route corrispondenti non sono registrate in `routes.go`. Bisogna collegarle e verificare che funzionino.

## Sotto-task

- [ ] Registrare le route in `routes.go`:
  - `POST /api/projects/:id/share` -> `SharedProjectHandler`
  - `DELETE /api/projects/:id/share` -> `UnshareProjectHandler`
- [ ] Verificare che gli handler gestiscano correttamente gli errori (utente non trovato, progetto gia condiviso, ecc.)
- [ ] Scrivere test unitari per `SharedProjectHandler`
- [ ] Scrivere test unitari per `UnshareProjectHandler`
- [ ] Test manuale con curl/httpie per validare il flusso end-to-end

## File coinvolti

- `internal/server/routes.go` — aggiungere 2 route
- `internal/server/handlers.go` — handler gia presenti, eventuale refine
- `internal/server/handlers_test.go` — test

## Dipendenze

- Nessuna — handler e query gia implementati

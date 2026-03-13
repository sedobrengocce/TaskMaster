# 07 - CORS Hardening per Produzione

## Descrizione
Attualmente il CORS e configurato con `AllowOrigins: ["*"]` in `server.go`. Per la produzione bisogna restringere le origini consentite e rendere la configurazione dinamica tramite variabili d'ambiente.

## Sotto-task

- [ ] Aggiungere variabile d'ambiente `CORS_ALLOWED_ORIGINS` (comma-separated)
- [ ] Aggiornare `internal/env/env.go` per leggere la nuova variabile
- [ ] Modificare la configurazione CORS in `internal/server/server.go`:
  - In sviluppo: `*` (default se variabile non impostata)
  - In produzione: lista esplicita di origini
- [ ] Restringere `AllowMethods` ai soli metodi usati (GET, POST, PUT, DELETE, OPTIONS)
- [ ] Restringere `AllowHeaders` ai soli header necessari (Authorization, Content-Type, X-CSRF-Token)
- [ ] Aggiungere `MaxAge` per il preflight caching (es. 3600 secondi)
- [ ] Aggiornare `.env.sample` con esempio della variabile
- [ ] Scrivere test per verificare che le origini vengano parsate correttamente

## File coinvolti

- `internal/server/server.go` — configurazione CORS
- `internal/env/env.go` — nuova variabile d'ambiente
- `.env.sample` — documentazione
- `internal/server/server_test.go` — test

## Dipendenze

- Nessuna — puo essere fatto indipendentemente

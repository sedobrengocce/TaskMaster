# 08 - CLI con Cobra + Viper

## Descrizione
Creare un'interfaccia a riga di comando per interagire con il server TaskMaster. La CLI comunichera con l'API REST e salvera il token di autenticazione localmente. Usare Cobra per il parsing dei comandi e Viper per la configurazione.

## Sotto-task

### Setup iniziale
- [ ] Aggiungere dipendenze: `go get github.com/spf13/cobra github.com/spf13/viper`
- [ ] Creare `cmd/cli/main.go` come entry point
- [ ] Creare `cmd/cli/root.go` con il comando root e configurazione Viper
- [ ] Configurare Viper per leggere da `~/.taskmaster/config.yaml`:
  - `server_url` — URL del server (default: `http://localhost:3000`)
  - `token` — JWT token salvato dopo login

### Comandi di autenticazione
- [ ] `taskmaster login` — richiede email e password, salva token
- [ ] `taskmaster logout` — revoca token e rimuove dal config
- [ ] `taskmaster register` — registra nuovo utente

### Comandi progetti
- [ ] `taskmaster projects list` — lista progetti
- [ ] `taskmaster projects create <name> [--color #hex]` — crea progetto
- [ ] `taskmaster projects delete <id>` — elimina progetto
- [ ] `taskmaster projects share <id> <user_id>` — condividi progetto

### Comandi task
- [ ] `taskmaster tasks list [--project <id>]` — lista task (filtro opzionale per progetto)
- [ ] `taskmaster tasks create <title> [--project <id>] [--type single|repetitive] [--priority N]` — crea task
- [ ] `taskmaster tasks complete <id>` — segna come completato
- [ ] `taskmaster tasks uncomplete <id>` — rimuovi completamento
- [ ] `taskmaster tasks delete <id>` — elimina task

### Vista settimanale
- [ ] `taskmaster weekly [--week 2026-03-09]` — mostra vista settimanale con tabella formattata

### Infrastruttura
- [ ] Creare package `cmd/cli/client/` con HTTP client wrapper per le API
- [ ] Gestire errori di rete e risposte non-200
- [ ] Gestire refresh token automatico quando il JWT scade
- [ ] Aggiornare `Makefile` con target `build-cli`

## File coinvolti

- `cmd/cli/` — nuova directory (tutti i file)
- `cmd/cli/main.go` — entry point
- `cmd/cli/root.go` — comando root + Viper
- `cmd/cli/client/` — HTTP client per le API
- `cmd/cli/commands/` — file per gruppo di comandi (auth, projects, tasks, weekly)
- `Makefile` — target di build
- `go.mod` — nuove dipendenze

## Dipendenze

- `01-task-crud-handlers.md` — servono le API dei task
- `02-task-completion-logging.md` — serve per il completamento
- `06-weekly-view.md` — serve per la vista settimanale
